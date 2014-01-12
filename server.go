package sshttp

import (
  "time"
  "fmt"
  "sync"
  "io"
  "bufio"
  "net"
  "net/http"
  "code.google.com/p/gosshnew/ssh"
)

type Server struct {
  Addr           string
  SSHConfig      *ssh.ServerConfig
  Handler        http.Handler
  ErrorHandler   func(error)

  wg             sync.WaitGroup
  accept         bool
}

func (srv *Server) ListenAndServe() error {
  addr := srv.Addr
  if addr == "" {
    addr = ":2280"
  }
  l, e := net.Listen("tcp", addr)
  if e != nil {
    return e
  }
  return srv.Serve(l)
}

func tempError(err error) bool {
  ne, ok := err.(net.Error)
  return ok && ne.Temporary()
}

func (srv *Server) handleError(err error) bool{
  if tempError(err) {
    time.Sleep(2*time.Millisecond)
    return false
  }else{
    if srv.ErrorHandler != nil {
      srv.ErrorHandler(err)
    }
    return true
  }
}

func (srv *Server) Serve(l net.Listener) error {
  srv.wg.Add(1)
  defer func(){
    l.Close()
    srv.wg.Done()
  }()
  return srv.ServeNoClose(l)
}

func (srv *Server) ServeNoClose(l net.Listener) error {
  if srv.Handler == nil {
    srv.Handler = http.DefaultServeMux
  }
  srv.accept = true
  for{
    con, err := l.Accept()
    if err != nil {
      if !tempError(err) {
        return err
      }
    }else{
      go srv.newClient(con)
    }
  }
  return nil
}

func (srv *Server) Shutdown() {
  srv.accept = false
  srv.wg.Wait()
}

func (srv *Server) newClient(c net.Conn){
  defer c.Close()
  con, chch, reqch, err := ssh.NewServerConn(c, srv.SSHConfig)
  defer con.Close()
  if err != nil {
    srv.handleError(err)
    return
  }
  srv.wg.Add(1)
  defer srv.wg.Done()
  for{
    select {
    case ch := <-chch :
      if ch == nil {
        return
      }else if srv.accept {
        go srv.newChannel(ch)
      }else{
        go srv.rejectChannel(ch)
      }
    case req := <-reqch :
      if req == nil {
        return
      }
    }
  }
}

func (srv *Server) rejectChannel(ch ssh.NewChannel){
  err := ch.Reject(ssh.Prohibited, "closed")
  if err != nil {
    srv.handleError(err)
  }
}

func (srv *Server) newChannel(nch ssh.NewChannel){
  var err error
  if nch.ChannelType() != "http" {
    err = nch.Reject(ssh.UnknownChannelType, "unknown channel type")
    if err != nil {
      srv.handleError(err)
    }
    return
  }
  ch, _, err := nch.Accept()
  defer ch.Close()
  rd := bufio.NewReader(ch)
  for{
    if !srv.accept {
      return
    }
    req, err := http.ReadRequest(rd)
    if err != nil {
      if srv.handleError(err) {
        return
      }else{
        continue
      }
    }
    srv.Handler.ServeHTTP(&sshResponseWriter{out: ch, header: make(http.Header)}, req)
  }
}

type sshResponseWriter struct{
  out io.ReadWriteCloser
  headerWritten bool
  header http.Header
}

func (w *sshResponseWriter) Header() http.Header {
  return w.header
}
func (w *sshResponseWriter) WriteHeader(status int) {
  if !w.headerWritten {
    io.WriteString(w.out, fmt.Sprintf("HTTP/1.1 %d %s\n", status, http.StatusText(status)))
    w.header.Write(w.out)
    io.WriteString(w.out,"\r\n\r\n")
    w.headerWritten = true
  }
}
func (w *sshResponseWriter) Write(b []byte) (int, error) {
  if !w.headerWritten {
    w.WriteHeader(http.StatusOK)
  }
  return w.out.Write(b)
}

