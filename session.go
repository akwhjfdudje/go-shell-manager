package main

import (
	"log"
	"net"
	"os"
	"io"
	"bufio"
	"fmt"
	"syscall"
	"os/signal"
)

type Session struct{
//structure for containing session information, such as the "id" of the session, io buffers, the listener and connection streams, 
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
}
// TODO : add multiple signal functionality

// Creates a new session, with provided id, ip address, and port to bind to
func NewSession(id int, ip string, port string) *Session{
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
	s.Listener, err = net.Listen("tcp", s.Ip + ":" + s.Port)
	if err != nil {
		fmt.Println("[!] Error binding: ", err)
	}
		// Accept the connection
		s.Conn, err = s.Listener.Accept()
		if err != nil {
			fmt.Println("[!] Error accepting connection: ", err)
		}	
}

// Interact method, to interact with the session
func (s *Session) Interact() {
	fmt.Println("[+] Interacting with session...")
	for{
		//Creates pipe to take output from connection and put in stdout
		s.Outread, s.Outwrite = io.Pipe()
		signal.Notify(s.External, syscall.SIGTSTP)	
		signal.Notify(s.External, syscall.SIGINT)
		for {
			//Handling SIGTSTP to close output to os.stdout
			go func(){
				select{
				case <- s.Bg:
					s.Outwrite.Close()
					s.Outread.Close()
					return
				default:
					io.Copy(s.Outwrite, s.Conn)	
				}
			} ()
			go func(){
				select{
				case <- s.Bg:
					s.Outread.Close()
					s.Outwrite.Close()
					return 
				default:
					io.Copy(os.Stdout,s.Outread)	
				}
			} ()		
			go s.CatchSignal()
			if s.Bg2{
				break
			}
			//Reads from stdin and sends to the connection	
			//fmt.Println("this line will be printed in an interact command")
			s.Input = bufio.NewReader(os.Stdin)
			cmd, err := s.Input.ReadString('\n')
			if err != nil{
				log.Fatal("Cannot send message: ", err)
			}
			s.Conn.Write([]byte(cmd))
		}
		s.Bg2 = false
		return
	}
} 

func (s *Session) CatchSignal(){
	//Catches any signal sent to s.External, and handles it
	for {
		select{
		case sig := <- s.External: 
			if sig == syscall.SIGTSTP {
				s.Bg <- true
				//s.Bg <- true
				s.Bg2 = true
				signal.Ignore(syscall.SIGTSTP)
				fmt.Println("[*] Ctrl-Z caught. Backgrounding current session...")
				return
			}
			if sig == syscall.SIGINT {
				fmt.Println("[?] Ctrl-C / exit caught.")
				signal.Ignore(syscall.SIGINT)
				return
			}
		default:
			break
		}
	}
}

