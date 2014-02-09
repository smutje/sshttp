package sshttp

import (
  "fmt"
  "io"
  "bufio"
  "net"
  "net/http"
  "code.google.com/p/gosshnew/ssh"
)

func HttpHandler(h http.Handler) ChannelHandler {
  return ChannelHandler(func(srv ServerChannel, nch ssh.NewChannel){
    httpChannel(h, srv, nch)
  })
}

func HttpHandlerFunc(f func(wr http.ResponseWriter,req *http.Request)) ChannelHandler {
  return HttpHandler(http.HandlerFunc(f))
}

func httpChannel(h http.Handler, srv ServerChannel, nch ssh.NewChannel){
  ch, reqch, err := nch.Accept()
  if err != nil {
    srv.HandleError(err)
    return
  }
  go ssh.DiscardRequests(reqch)
  defer ch.Close()
  rd := bufio.NewReader(ch)
  for{
    if !srv.Accept() {
      return
    }
    req, err := http.ReadRequest(rd)
    if err != nil {
      if srv.HandleError(err) {
        return
      }else{
        continue
      }
    }
    h.ServeHTTP(&sshResponseWriter{out: ch, header: make(http.Header)}, req)
  }
}

type sshResponseWriter struct{
  out io.ReadWriteCloser
  headerWritten bool
  header http.Header
}

func (w *sshResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
  return nil,nil,nil
}
func (w *sshResponseWriter) Header() http.Header {
  return w.header
}
func (w *sshResponseWriter) WriteHeader(status int) {
  if !w.headerWritten {
    io.WriteString(w.out, fmt.Sprintf("HTTP/1.1 %d %s\n", status, http.StatusText(status)))
    w.header.Write(w.out)
    io.WriteString(w.out,"\r\n")
    w.headerWritten = true
  }
}
func (w *sshResponseWriter) Write(b []byte) (int, error) {
  if !w.headerWritten {
    w.WriteHeader(http.StatusOK)
  }
  return w.out.Write(b)
}
