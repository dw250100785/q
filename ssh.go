package q

// Minimal managment over SSH for your Q web server
//
// Declaration:
//
// q.Q{
//   Host:         "localhost:80",
// 	DevMode:       true,
// 	SSH:           q.SSH{Host: "localhost:8888", KeyPath: "./q_rsa_generate_if_not_exists", Users: q.Users{"kataras": []byte("pass")}},
// 	// other fields here...
// }.Go()
//
// Usage:
//
// ssh kataras@localhost -p 8888 help
//
// Commands available:
//
// stop
// start
// restart
// help

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/kataras/q/errors"
	"golang.org/x/crypto/ssh"
)

var (
	// SSHBanner is the banner goes on top of the 'ssh help message'
	// it can be changed, defaults is the Q's banner
	SSHBanner = banner

	helpMessage = SSHBanner + `
COMMANDS:
	  {{ range $index, $cmd := .Commands }}
		  {{- $cmd.Name }} - {{ $cmd.Description }}
	  {{ end }}
USAGE:
	  ssh myusername@{{ .Hostname}} -p {{ .Port }} {{ first .Commands}}
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

/* these goes to Q instance's builder
var defaultCommands = Commands{
	Command{Name: "stop", Description: "Stops the http server", Action: func(conn ssh.Channel) {

	}},
	Command{Name: "start", Description: "", Action: func(conn ssh.Channel) {

	}},
	Command{Name: "restart", Description: "", Action: func(conn ssh.Channel) {

	}},
}
*/

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

func generateSigner(keypath string) (ssh.Signer, error) {
	if keypath == "" {
		keypath = DefaultSSHKeyPath
	}

	if !directoryExists(keypath) {
		os.MkdirAll(filepath.Dir(keypath), os.ModePerm)
		keygenCmd := exec.Command("ssh-keygen", "-f", keypath, "-t", "rsa", "-N", "")
		_, err := keygenCmd.Output()
		if err != nil {
			return nil, err
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

func parsePayload(payloadB []byte, prefix string) (string, error) {
	payload := string(payloadB)
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
// ----------------------------------SSH implementation---------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

// SSH : Simple SSH interface for Q web framework, does not implements the most secure options and code,
// but its should works
// use it at your own risk.
type SSH struct {
	KeyPath  string // C:/Users/kataras/.ssh/q_rsa
	Host     string // host:port
	listener net.Listener
	Users    Users    // map[string]string{ "username":[]byte("password"), "my_second_username" : []byte("my_second_password")}
	Commands Commands // Commands{Command{Name: "restart", Description:"restarts & rebuild the server", Action: func(ssh.Channel){}}}
	// note for Commands field:
	// the default  Q's commands are defined inside the q.go file, I tried to make this file as standalone as I can, because it will be used for Iris web framework also.
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

func (s *SSH) writeHelp(wr io.Writer) {
	port := parsePort(s.Host)
	hostname := parseHostname(s.Host)

	data := map[string]interface{}{
		"Hostname": hostname, "Port": port,
		"Commands": s.Commands,
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
	privateKey, err := generateSigner(s.KeyPath)
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
			//log.Println("Session closed")
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
					// comes after pty-req
					return
				}

			case "exec":
				{
					payload, perr := parsePayload(req.Payload, "")
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

// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------
// ----------------------------------Q+SSH----------------------------------------------
// -------------------------------------------------------------------------------------
// -------------------------------------------------------------------------------------

func (s *SSH) bindTo(q *Q) {
	if s.Enabled() && !s.IsListening() { // check if not listening because on restart this block will re-executing,but we don't want to start ssh again, ssh will never stops.

		if q.DevMode && s.Logger == nil {
			s.Logger = q.Logger
		}

		sshCommands := Commands{
			// Note for stop If you have opened a tab with Q route:
			//  in order to see that the http listener has closed you have to close your browser and re-navigate(browsers caches the tcp connection)
			Command{Name: "stop", Description: "Stops the Q http server", Action: func(conn ssh.Channel) {
				if q.listener != nil {
					q.listener.Close()
					q.listener = nil
					conn.Write([]byte("Server has stopped"))
				} else {
					conn.Write([]byte("Error: Server is not even builded yet!"))
				}
			}},
			Command{Name: "start", Description: "Starts the Q http server", Action: func(conn ssh.Channel) {
				if q.listener == nil {
					go q.runServer()
				}
				conn.Write([]byte("Server has started"))
			}},
			Command{Name: "restart", Description: "Restarts the Q http server. Note: It doesn't re-build the whole Q for security reasons.", Action: func(conn ssh.Channel) {
				if q.listener != nil {
					q.listener.Close()
					q.listener = nil
				}
				go q.runServer()
				conn.Write([]byte("Server has restarted"))
			}},
		}

		for _, cmd := range sshCommands {
			if _, found := s.Commands.ByName(cmd.Name); !found {
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
