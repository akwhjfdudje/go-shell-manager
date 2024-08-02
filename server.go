package main
import(
	"fmt"
	"strings"
	"net/http"
	"net"
	"strconv"
	"log"
	"bufio"
	"os"
	"regexp"

)
//Global variables for handling total sessions and killing servers
var sessions []*Session
var count int = len(sessions)
var killServer chan bool
var startedServer bool

func GetIpFromInt(intrf string) (string){
	ipv4Regex := `^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`
	ipv4Pattern := regexp.MustCompile(ipv4Regex)
	if ipv4Pattern.MatchString(intrf){
		return intrf
	}
	ief, err := net.InterfaceByName(intrf)
    if err !=nil{
            fmt.Println("[!] Error getting interface: ",err)
			return ""
    }
    addrs, err := ief.Addrs()
    if err !=nil{
            fmt.Println("[!] Error getting addresses: ", err)
			return ""
    }

    tcpAddr := &net.TCPAddr{
        IP: addrs[0].(*net.IPNet).IP,
    }
	return tcpAddr.IP.String()
}

func prompt(){
	//Prompt function to handle user commands
	var userinput string
	fmt.Print("[sessions : ", count, "]> ")
	reader := bufio.NewReader(os.Stdin)
    	userinput, err := reader.ReadString('\n')
    	if err != nil {
        	log.Println("[!] Problem getting user input: ", err)
    	}
	userinput = strings.TrimSuffix(userinput, "\n")
	u := strings.Fields(userinput)
	if len(u) == 0{
		prompt()
	}
	switch u[0]{
	case "":	
	case "listen":
		if len(u) < 3{
			fmt.Println("[!] Usage: listen <ip or interface> <port>")
			prompt()
		}
		ip := GetIpFromInt(u[1])
		session := NewSession(count, ip, u[2])
		sessions = append(sessions, session)
		count = len(sessions)
		session.Listen()
		session.Interact()
	case "serve":
		if len(u) < 3{
			fmt.Println("[!] Usage: serve <ip or interface> <port>")
			prompt()
		}
		ip := GetIpFromInt(u[1])
		go serve(ip, u[2])	
	case "kill":	
		if !startedServer{
			fmt.Println("[!] No servers are running")
			prompt()
			}
		killServer <- true
	case "interact":
		if len(u) < 2{
			fmt.Println("[!] Usage: interact <session_id>")
			prompt()
		}
		id, err := strconv.Atoi(u[1])
		if err != nil {
			fmt.Println("[!] Error in getting id: ", err)
			prompt()
		}
		session := sessions[id]
		session.Interact()
	case "help":
		fmt.Println("[?] Available commands:\n    listen: listens on a specified port and IP\n    serve: starts an HTTP server\n    interact: interacts with a session\n    exit: exits the program")
	case "exit":
		fmt.Println("[*] Exiting program...")
		os.Exit(0)
	default:
		fmt.Println("[!] Unknown command: ", userinput)
	}
	prompt()
	//TODO - Add error handling to this function, goes without saying that error handling has to be added to the whole codebase
}

func serve(ip string, port string) {
	//Creates Server instance, then starts an HTTP file server
	killServer = make(chan bool)
	fmt.Println("[+] Starting HTTP server on " + ip + ":" + port)
	fileServer := http.FileServer(http.Dir("."))
	server := &http.Server{
		Addr: ip + ":" + port,
		Handler: fileServer,
	}
	go func() {
		server.ListenAndServe()
	} ()
	startedServer = true
	//Handle closing the server
	<- killServer
	server.Close()			
	fmt.Println("[*] Closed server.")	
}
func NewSession(id int, ip string, port string) *Session{
	fmt.Println("[+] Creating new session...")		
	return &Session{
		Id: id,
		Port: port,
		Ip: ip,
		External: make(chan os.Signal, 1),
		Bg: make(chan bool, 1),
		//breaking change, will need to see if buffered channels are neccessary

	}	
}

func main(){
	fmt.Println("----Shell Manager----")
	prompt()
}

