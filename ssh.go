package q

// Minimal managment over SSH for your Q web server
//
// Declaration:
//
// q.Q{
//   Host:         "localhost:80",
// 	DevMode:       true,
// 	SSH:           q.SSH{Host: "localhost:22", KeyPath: "./q_rsa_generate_if_not_exists", Users: q.Users{"kataras": []byte("pass")}},
// 	// other fields here...
// }.Go()
//
// Usage:
// via interactive command shell:
//
// $ ssh kataras@localhost
//
// or via standalone command and exit:
//
// $ ssh kataras@localhost stop
//
//
// Commands available:
//
// stop
// start
// restart
// log
// help
// exit

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/kardianos/osext"
	"github.com/kardianos/service"
	"github.com/kataras/q/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------Q+SSH----------------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

func _output(format string, a ...interface{}) func(io.Writer) {
	if format[len(format)-3:] != "\n" {
		format += "\n"
	}
	msgBytes := []byte(fmt.Sprintf(format, a...))

	return func(w io.Writer) {
		w.Write(msgBytes)
	}
}

type systemServiceWrapper struct{}

func (w *systemServiceWrapper) Start(s service.Service) error {
	return nil
}

func (w *systemServiceWrapper) Stop(s service.Service) error {
	return nil
}

func (s *SSH) bindTo(q *Q) {
	if s.Enabled() && !s.IsListening() { // check if not listening because on restart this block will re-executing,but we don't want to start ssh again, ssh will never stops.

		if q.DevMode && s.Logger == nil {
			s.Logger = q.Logger
		}

		// cache the messages to be sent to the channel, no need to produce memory allocations here
		statusRunningMsg := _output("The HTTP Server is running.")
		statusNotRunningMsg := _output("The HTTP Server is NOT running. ")

		serverStoppedMsg := _output("The HTTP Server has been stopped.")
		errServerNotReadyMsg := _output("Error: HTTP Server is not even builded yet!")

		serverStartedMsg := _output("The HTTP Server has been started.")
		serverRestartedMsg := _output("The HTTP Server has been restarted.")

		loggerStartedMsg := _output("Logger has been registered to the HTTP Server.\nNew Requests will be printed here.\nYou can still type 'exit' to close this SSH Session.\n\n")
		//

		sshCommands := Commands{
			Command{Name: "status", Description: "Prompts the status of the HTTP Server, is listening(started) or not(stopped).", Action: func(conn ssh.Channel) {
				if q.listener != nil {
					statusRunningMsg(conn)
				} else {
					statusNotRunningMsg(conn)
				}
				execPath, err := osext.Executable() // this works fine, if the developer builded the go app, if just go run main.go then prints the temporary path which the go tool creates
				if err == nil {
					conn.Write([]byte("[EXEC] " + execPath + "\n"))
				}
			}},
			// Note for stop If you have opened a tab with Q route:
			//  in order to see that the http listener has closed you have to close your browser and re-navigate(browsers caches the tcp connection)
			Command{Name: "stop", Description: "Stops the HTTP Server.", Action: func(conn ssh.Channel) {
				if q.listener != nil {
					q.listener.Close()
					q.listener = nil
					serverStoppedMsg(conn)
				} else {
					errServerNotReadyMsg(conn)
				}
			}},
			Command{Name: "start", Description: "Starts the HTTP Server.", Action: func(conn ssh.Channel) {
				if q.listener == nil {
					go q.runServer()
				}
				serverStartedMsg(conn)
			}},
			Command{Name: "restart", Description: "Restarts the HTTP Server.", Action: func(conn ssh.Channel) {
				if q.listener != nil {
					q.listener.Close()
					q.listener = nil
				}
				go q.runServer()
				serverRestartedMsg(conn)
			}},
			Command{Name: "service", Description: "[REQUIRES HTTP SERVER's ADMIN PRIVILEGE] Adds the web server to the system services, use it when you want to make your server to autorun on reboot", Action: func(conn ssh.Channel) {
				///TODO:
				// 1. Unistall service and change the 'service' to 'install service'
				// 2. Fix, this current implementation doesn't works on windows 10 it says that the service is not responding to request and start...
				// 2.1 the fix is maybe add these and change the s.Install to s.Run to the $DESKTOP/some/q/main.go I will try this
				// as the example shows:
				/*
					package main

					import (
						"flag"
						"log"
						"time"

						"github.com/kardianos/service"
					)

					var logger service.Logger

					// Program structures.
					//  Define Start and Stop methods.
					type program struct {
						exit chan struct{}
					}

					func (p *program) Start(s service.Service) error {
						if service.Interactive() {
							logger.Info("Running in terminal.")
						} else {
							logger.Info("Running under service manager.")
						}
						p.exit = make(chan struct{})

						// Start should not block. Do the actual work async.
						go p.run()
						return nil
					}
					func (p *program) run() error {
						logger.Infof("I'm running %v.", service.Platform())
						ticker := time.NewTicker(2 * time.Second)
						for {
							select {
							case tm := <-ticker.C:
								logger.Infof("Still running at %v...", tm)
							case <-p.exit:
								ticker.Stop()
								return nil
							}
						}
					}
					func (p *program) Stop(s service.Service) error {
						// Any work in Stop should be quick, usually a few seconds at most.
						logger.Info("I'm Stopping!")
						close(p.exit)
						return nil
					}

					// Service setup.
					//   Define service config.
					//   Create the service.
					//   Setup the logger.
					//   Handle service controls (optional).
					//   Run the service.
					func main() {
						svcFlag := flag.String("service", "", "Control the system service.")
						flag.Parse()

						svcConfig := &service.Config{
							Name:        "GoServiceExampleLogging",
							DisplayName: "Go Service Example for Logging",
							Description: "This is an example Go service that outputs log messages.",
						}

						prg := &program{}
						s, err := service.New(prg, svcConfig)
						if err != nil {
							log.Fatal(err)
						}
						errs := make(chan error, 5)
						logger, err = s.Logger(errs)
						if err != nil {
							log.Fatal(err)
						}

						go func() {
							for {
								err := <-errs
								if err != nil {
									log.Print(err)
								}
							}
						}()

						if len(*svcFlag) != 0 {
							err := service.Control(s, *svcFlag)
							if err != nil {
								log.Printf("Valid actions: %q\n", service.ControlAction)
								log.Fatal(err)
							}
							return
						}
						err = s.Run()
						if err != nil {
							logger.Error(err)
						}
					}


				*/
				// remember: run command line as administrator > sc delete "Q Web Server - $DATETIME" to delete the service, do it on each test.
				svcConfig := &service.Config{
					Name:        "Q Web Server - " + time.Now().Format(q.TimeFormat),
					DisplayName: "Q Web Server - " + time.Now().Format(q.TimeFormat),
					Description: "The web server which has been registered by SSH interface.",
				}

				prg := &systemServiceWrapper{}
				s, err := service.New(prg, svcConfig)

				if err != nil {
					conn.Write([]byte(err.Error() + "\n"))
					return
				}

				err = s.Install()
				if err != nil {
					conn.Write([]byte(err.Error() + "\n"))
					return
				}
				conn.Write([]byte("Service has been registered.\n"))

				/*
					_, err = s.Logger(nil)
					if err != nil {
						conn.Write([]byte(err.Error()))
					}
					err = s.Run()
					if err != nil {
						conn.Write([]byte(err.Error()))
					}*/

			}},
			Command{Name: "log", Description: "Adds a logger to the HTTP Server, waits for requests and prints them here.", Action: func(conn ssh.Channel) {
				// the ssh user can still write commands, this is not blocking anything.
				loggerMiddleware := Handlers{NewLoggerHandler(conn, true)}
				for _, r := range q.Request.mux.lookups {
					r.handlers = append(loggerMiddleware, r.handlers...)
				}

				// register to the errors also
				errorLoggerHandler := NewLoggerHandler(conn, false)
				for k, v := range q.Request.Errors {
					errorH := v
					// wrap the error handler with the ssh logger middleware
					q.Request.Errors[k] = func(ctx *Context) {
						errorH(ctx)
						errorLoggerHandler(ctx) // after the error handler because that is setting the status code.
					}
				}

				q.Request.mux.build() // rebuilds the mux, this will give some milleseconds of downtime

				loggerStartedMsg(conn)
				// the middleware will still to run, we could remove it on exit but exit is general command I dont want to touch that
				// we could make a command like 'log stop' or on 'stop' to remove the middleware...I will think about it.
			}},
		}

		for _, cmd := range sshCommands {
			if _, found := s.Commands.ByName(cmd.Name); !found { // yes, the user can add custom commands too, I will cover this on docs some day, it's not too hard if you see the code.
				s.Commands.Add(cmd)
			}
		}

		if !q.DisableServer {
			go func() {
				q.must(s.Listen())
			}()
		}
	}
}

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------SSH implementation---------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

var (
	// SSHBanner is the banner goes on top of the 'ssh help message'
	// it can be changed, defaults is the Q's banner
	SSHBanner = banner

	helpMessage = SSHBanner + `
COMMANDS:
	  {{ range $index, $cmd := .Commands }}
		  {{- $cmd.Name }} | {{ $cmd.Description }}
	  {{ end }}
USAGE:
	  ssh myusername@{{ .Hostname}} {{ .PortDeclaration }} {{ first .Commands}}
	  or just write the command below
VERSION:
	  {{ .Version }}

`
	helpTmpl *template.Template
)

func init() {
	var err error
	helpTmpl = template.New("help_message").Funcs(template.FuncMap{"first": func(cmds Commands) string {
		if len(cmds) > 0 {
			return cmds[0].Name
		}

		return ""
	}})

	helpTmpl, err = helpTmpl.Parse(helpMessage)
	if err != nil {
		panic(err.Error())
	}
}

//no need of SSH prefix on these types, we don't have other commands
// use of struct and no global variables because we want each Q instance to have its own SSH interface.

// Action the command's handler
type Action func(ssh.Channel)

// Command contains the registered SSH commands
// contains a Name which is the payload string
// Description which is the description of the command shows to the admin/user
// Action is the particular command's handler
type Command struct {
	Name        string
	Description string
	Action      Action
}

// Commands the SSH Commands, it's just a type of []Command
type Commands []Command

// Add adds command(s) to the commands list
func (c *Commands) Add(cmd ...Command) {
	pCommands := *c
	*c = append(pCommands, cmd...)
}

// ByName returns the command by its Name
// if not found returns a zero-value Command and false as the second output parameter.
func (c *Commands) ByName(commandName string) (cmd Command, found bool) {
	pCommands := *c
	for _, cmd = range pCommands {
		if cmd.Name == commandName {
			found = true
			return
		}
	}
	return
}

// Users SSH.Users field, it's just map[string][]byte (username:password)
type Users map[string][]byte

func (m Users) exists(username string, pass []byte) bool {
	for k, v := range m {
		if k == username && bytes.Equal(v, pass) {
			return true
		}
	}
	return false

}

// DefaultSSHKeyPath used if SSH.KeyPath is empty. Defaults to: "q_rsa". It can be changed.
var DefaultSSHKeyPath = "q_rsa"

var errSSHExecutableNotFound = errors.New(`Cannot generate ssh private key: ssh-keygen couldn't be found. Please specify the ssh[.exe] and ssh-keygen[.exe]
  path on your operating system's environment's $PATH or set the configuration field 'Bin'.\n For example, on windows, the path is: C:\\Program Files\\Git\usr\\bin. Error Trace: %q`)

func generateSigner(keypath string, sshKeygenBin string) (ssh.Signer, error) {
	if keypath == "" {
		keypath = DefaultSSHKeyPath
	}
	if sshKeygenBin != "" {
		// if empty then the user should specify the ssh-keygen bin path (if not setted already)
		// on the $PATH system environment, otherwise it will panic.
		if sshKeygenBin[len(sshKeygenBin)-1] != os.PathSeparator {
			sshKeygenBin += string(os.PathSeparator)
		}
		sshKeygenBin += "ssh-keygen"
		if isWindows {
			sshKeygenBin += ".exe"
		}
	} else {
		sshKeygenBin = "ssh-keygen"
	}
	if !directoryExists(keypath) {
		os.MkdirAll(filepath.Dir(keypath), os.ModePerm)
		keygenCmd := exec.Command(sshKeygenBin, "-f", keypath, "-t", "rsa", "-N", "")
		_, err := keygenCmd.Output()
		if err != nil {
			panic(errSSHExecutableNotFound.Format(err.Error()))
		}
	}

	pemBytes, err := ioutil.ReadFile(keypath)
	if err != nil {
		return nil, err
	}

	return ssh.ParsePrivateKey(pemBytes)
}

func validChannel(ch ssh.NewChannel) bool {
	if typ := ch.ChannelType(); typ != "session" {
		ch.Reject(ssh.UnknownChannelType, typ)
		return false
	}
	return true
}

func execCmd(cmd *exec.Cmd, ch ssh.Channel) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	input, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	go io.Copy(input, ch)
	io.Copy(ch, stdout)
	io.Copy(ch.Stderr(), stderr)

	if err = cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func sendExitStatus(ch ssh.Channel) {
	ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
}

var errInvalidSSHCommand = errors.New("Invalid Command: '%s'")

func parsePayload(payload string, prefix string) (string, error) {
	payloadUTF8 := strings.Map(func(r rune) rune {
		if r >= 32 && r < 127 {
			return r
		}
		return -1
	}, payload)

	if prefIdx := strings.Index(payloadUTF8, prefix); prefIdx != -1 {
		p := strings.TrimSpace(payloadUTF8[prefIdx+len(prefix):])
		return p, nil
	}
	return "", errInvalidSSHCommand.Format(payload)
}

const (
	isWindows = runtime.GOOS == "windows"
	isMac     = runtime.GOOS == "darwin"
)

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------SSH Server-----------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// SSH : Simple SSH interface for Q web framework, does not implements the most secure options and code,
// but its should works
// use it at your own risk.
type SSH struct {
	Bin      string // windows: C:/Program Files/Git/usr/bin, it's the ssh[.exe] and ssh-keygen[.exe], we only need the ssh-keygen.
	KeyPath  string // C:/Users/kataras/.ssh/q_rsa
	Host     string // host:port
	listener net.Listener
	Users    Users    // map[string][]byte]{ "username":[]byte("password"), "my_second_username" : []byte("my_second_password")}
	Commands Commands // Commands{Command{Name: "restart", Description:"restarts & rebuild the server", Action: func(ssh.Channel){}}}
	// note for Commands field:
	// the default  Q's commands are defined at the end of this file, I tried to make this file as standalone as I can, because it will be used for Iris web framework also.
	Shell  bool        // Set it to true to enable execute terminal's commands(system commands) via ssh if no other command is found from the Commands field. Defaults to false for security reasons
	Logger *log.Logger // log.New(...)/ $qinstance.Logger, fill it when you want to receive debug and info/warnings messages
}

// Enabled returns true if SSH can be started, if Host != ""
func (s SSH) Enabled() bool {
	return s.Host != ""
}

// IsListening returns true if ssh server has been started
func (s SSH) IsListening() bool {
	return s.Enabled() && s.listener != nil
}

func (s *SSH) logf(format string, a ...interface{}) {
	if s.Logger != nil {
		s.Logger.Printf(format, a...)
	}
}

// commands that exists on all ssh interfaces, both Q and Iris(future)
var standardCommands = Commands{Command{Name: "help", Description: "Opens up the assistance"},
	Command{Name: "exit", Description: "Exits from the terminal (if interactive shell)"}}

// parsePort receives an addr of form host[:port] and returns the port part of it
// ex: localhost:8080 will return the `8080`, mydomain.com will return the '22'
func parseSSHPort(addr string) int {
	if portIdx := strings.IndexByte(addr, ':'); portIdx != -1 {
		afP := addr[portIdx+1:]
		p, err := strconv.Atoi(afP)
		if err == nil {
			return p
		}
	}
	return 22
}

func (s *SSH) writeHelp(wr io.Writer) {
	port := parseSSHPort(s.Host)
	hostname := parseHostname(s.Host)

	data := map[string]interface{}{
		"Hostname": hostname, "PortDeclaration": "-p " + strconv.Itoa(port),
		"Commands": append(s.Commands, standardCommands...),
		"Version":  Version,
	}

	helpTmpl.Execute(wr, data)
}

var (
	errUserInvalid  = errors.New("Username or Password rejected for: %q")
	errServerListen = errors.New("Cannot listen to: %s, Trace: %s")
)

// Listen starts the SSH Server
func (s *SSH) Listen() error {

	// get the key
	privateKey, err := generateSigner(s.KeyPath, s.Bin)
	if err != nil {
		return err
	}
	// prepare the server's configuration
	cfg := &ssh.ServerConfig{
		// NoClientAuth: true to allow anyone to login, nooo
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			username := c.User()
			if !s.Users.exists(username, pass) {
				return nil, errUserInvalid.Format(username)
			}
			return nil, nil
		}}

	cfg.AddHostKey(privateKey)

	// start the server with the configuration we just made.
	var lerr error
	s.listener, lerr = net.Listen("tcp", s.Host)
	if lerr != nil {
		return errServerListen.Format(s.Host, lerr.Error())
	}

	// ready to accept incoming requests
	s.logf("SSH Server is running")
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.logf(err.Error())
			continue
		}
		// handshake first

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
		if err != nil {
			s.logf(err.Error())
			continue
		}

		s.logf("New SSH Connection has been enstablish from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

		// discard all global requests
		go ssh.DiscardRequests(reqs)
		// accept all current chanels
		go s.handleChannels(chans)
	}

}

func (s *SSH) handleChannels(chans <-chan ssh.NewChannel) {
	for ch := range chans {
		go s.handleChannel(ch)
	}
}

var errUnsupportedReqType = errors.New("Unsupported request type: %q")

func (s *SSH) handleChannel(newChannel ssh.NewChannel) {
	// we working from terminal, so only type of "session" is allowed.
	if !validChannel(newChannel) {
		return
	}

	conn, reqs, err := newChannel.Accept()
	if err != nil {
		s.logf(err.Error())
		return
	}

	go func(in <-chan *ssh.Request) {
		defer func() {
			conn.Close()
			//debug
			s.logf("Session closed")
		}()

		for req := range in {
			var err error
			defer func() {
				if err != nil {
					conn.Write([]byte(err.Error()))
				}
				sendExitStatus(conn)
			}()

			switch req.Type {
			case "pty-req":
				{
					s.writeHelp(conn)
					req.Reply(true, nil)
				}

			case "shell":
				{
					// comes after pty-req, this is when the user just use this form: ssh kataras@mydomain.com -p 22
					// then we want interactive shell which will execute the commands:
					term := terminal.NewTerminal(conn, "> ")

					for {
						line, lerr := term.ReadLine()
						if lerr == io.EOF {
							return
						}
						if lerr != nil {
							err = lerr
							s.logf(lerr.Error())
							continue
						}

						payload, perr := parsePayload(line, "")
						if perr != nil {
							err = perr
							return
						}

						if payload == "help" {
							s.writeHelp(conn)
							continue
						} else if payload == "exit" {
							return
						}

						if cmd, found := s.Commands.ByName(payload); found {
							cmd.Action(conn)
						} else if s.Shell {
							// yes every time check that
							if isWindows {
								execCmd(exec.Command("cmd", "/C", payload), conn)
							} else {
								execCmd(exec.Command("sh", "-c", payload), conn)
							}
						} else {
							conn.Write([]byte(errInvalidSSHCommand.Format(payload).Error() + "\n"))
						}

						//s.logf(line)
					}
				}

			case "exec":
				{
					// this is the place which the user executed something like that: ssh kataras@mydomain.com -p 22 stop
					// a direct command, we don' t open the interactive shell, just execute the command and exit.
					payload, perr := parsePayload(string(req.Payload), "")
					if perr != nil {
						err = perr
						return
					}

					if cmd, found := s.Commands.ByName(payload); found {
						cmd.Action(conn)
					} else if payload == "help" {
						s.writeHelp(conn)
					} else if s.Shell {
						// yes every time check that
						if isWindows {
							execCmd(exec.Command("cmd", "/C", payload), conn)
						} else {
							execCmd(exec.Command("sh", "-c", payload), conn)
						}
					} else {
						err = errInvalidSSHCommand.Format(payload)
					}
					return
				}
			default:
				{
					err = errUnsupportedReqType.Format(req.Type)
					return
				}
			}
		}

	}(reqs)

}
