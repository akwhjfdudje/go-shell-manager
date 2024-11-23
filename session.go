package main

import (
	"log"
	//"time"
	"net"
	"os"
	"io"
	"bufio"
	"fmt"
	"syscall"
	"os/signal"
	"context"
)

// Session object for:
//	 containing session information, 
//	 such as the "id" of the session, 
//   io buffers, 
//   the listener and connection streams
type Session struct{
	Id int
	Port string
	Ip string
	Outread *io.PipeReader
	Outwrite *io.PipeWriter 
	Input *bufio.Reader
	Listener net.Listener
	Conn net.Conn
	External chan os.Signal
	Bg chan bool
	Bg2 bool
	Config net.ListenConfig
}
// TODO : add multiple signal functionality
// TODO : fix elements in Session struct to reduce redundancy

// Method to create config for current session
func (s *Session) CreateConfig() {
	// This code works, can reuse ports
	// TODO: manage when ports can't be reused
	s.Config = net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var OpErr error
			if err := c.Control( func( fd uintptr ) {
                OpErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				OpErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
            }); err != nil {
				fmt.Println("[!] Couldn't create configuration: ", err)
                return err
            }
            return OpErr
		},
	}
}

// Creates a new session, with provided id, ip address, and port to bind to
func NewSession(id int, ip string, port string) *Session{

	// Create pointer to a new session
	fmt.Println("[+] Creating new session...")		
	return &Session{
		Id: id,
		Port: port,
		Ip: ip,
		External: make(chan os.Signal, 1),
		Bg: make(chan bool, 1),
	}
}

// Listen method, to start the listener
func (s *Session) Listen() {
	// Start the TCP listener to listen on port "port"
	var err error
	fmt.Println("[+] Starting listener on " + s.Ip + ":" + s.Port + "...")
	s.CreateConfig()
	s.Listener, err = s.Config.Listen(context.Background(), "tcp", s.Ip + ":" + s.Port)
	if err != nil {
		fmt.Println("[!] Error binding: ", err)
		return
	}
	// Accept the connection
	s.Conn, err = s.Listener.Accept()
	if err != nil {
		fmt.Println("[!] Error accepting connection: ", err)
		return
	}	
}

// Method to close buffers for writing and reading
func (s *Session) KillBuffers() {
	s.Outwrite.Close()
	s.Outread.Close()
}

// Method to close connection
func (s *Session) CloseConnection() {
	err := s.Conn.Close()
	if err != nil {
		fmt.Println("[!] Connection closed with errors: ", err)
	}
}

// Method to handle system signals
func (s *Session) StartNotifiers() {
	signal.Notify(s.External, syscall.SIGTSTP)	
	signal.Notify(s.External, syscall.SIGINT)
}

// These four methods attempt to pipe input from stdin to connection,
// and back, from conn to stdout
func (s *Session) StartPipes() {
	//Creates pipe to take output from connection and put in stdout
	s.Outread, s.Outwrite = io.Pipe()
}

func (s *Session) GetOutputToStdout() {
	select{
	case <- s.Bg:
		s.KillBuffers()
		return 
	default:
		io.Copy(os.Stdout,s.Outread)	
	}
}

func (s *Session) GetOutputFromConn() {
	select{
	case <- s.Bg:
		s.KillBuffers()
		return 
	default:
		io.Copy(s.Outwrite, s.Conn)	
	}
}

func (s *Session) GetInputFromStdin() {
	//Reads from stdin and sends to the connection
	s.Input = bufio.NewReader(os.Stdin)
	cmd, err := s.Input.ReadString('\n')
	if err != nil{
		log.Fatal("Cannot send message: ", err)
	}
	s.Conn.Write([]byte(cmd))
}

// Interact method, to interact with the session
func (s *Session) Interact() {
	fmt.Println("[+] Interacting with session...")
	for{
		// Initializign pipes, signal handlers
		s.StartPipes()
		s.StartNotifiers()
		for {
			// Connecting pipes, handling signals
			go s.GetOutputFromConn()
			go s.GetOutputToStdout()		
			go s.CatchSignal()
			if s.Bg2{
				break
			}
			// Taking input
			s.GetInputFromStdin()
		}
		s.Bg2 = false
		return
	}
} 

// Method to kill session
func (s *Session) KillSession() {
	sessions[s.Id] = nil
}

// Function to check if a given ip:port is bound
func BadBind(ip string, port string) (bool) {
	ln, err := net.Listen("tcp", ip + ":" + port)
	if err != nil {
		fmt.Println("[!] Error binding: ", err)
		if ln != nil {
			ln.Close()
		}
		return true
	}
	defer ln.Close()
	return false
}

// Method to catch system signals
func (s *Session) CatchSignal(){
	//Catches any signal sent to s.External, and handles it
	for {
		select{
		case sig := <- s.External: 
			if sig == syscall.SIGTSTP {
				s.Bg <- true
				s.Bg2 = true
				signal.Ignore(syscall.SIGTSTP)
				fmt.Println("[*] Ctrl-Z caught. Backgrounding current session...")
				return
			}
			if sig == syscall.SIGINT {
				s.Bg <- true
				s.Bg2 = true
				s.KillBuffers()
				s.CloseConnection()
				//time.Sleep(2 * time.Second)
				s.KillSession()
				// New bugs, yay
				// TODO: implement SO_REUSEPORT to allow for port re-use
				count -= 1
				signal.Reset(syscall.SIGINT)
				// TODO: implement signal handling to prevent signals from being caught in the prompt
				fmt.Println("[?] Ctrl-C / exit caught.")
				return
			}
		default:
			break
		}
	}
}

