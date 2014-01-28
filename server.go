package sshttp

import (
  "time"
  "sync"
  "net"
  "code.google.com/p/gosshnew/ssh"
)

type ServerChannel interface {
  Accept() bool
  HandleError(error) bool
}

type ChannelHandler func(srv ServerChannel, ch ssh.NewChannel)

type Server struct {
  Addr           string
  SSHConfig      *ssh.ServerConfig
  Handler        ChannelHandler
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

func (srv *Server) HandleError(err error) bool{
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

func (srv *Server) Accept() bool {
  return srv.accept
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
    srv.HandleError(err)
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
        go srv.Handler(srv,ch)
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
    srv.HandleError(err)
  }
}

