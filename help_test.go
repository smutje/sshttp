package sshttp

import (
  "io"
  "bytes"
  "errors"
  "net"
  "time"
  "strings"
  "sync"
  "github.com/smutje/deadlinecond"
)

var (
  Closed = errors.New("closed")
)

type fakeAddr string

func (f *fakeAddr) Network() string {
  s := strings.SplitN(string(*f),"://",2)
  return s[0]
}

func (f *fakeAddr) String() string {
  return string(*f)
}

const (
  fa = fakeAddr("pipe://");
)

type syncBuffer struct {
  *deadlinecond.Cond
  buffer bytes.Buffer
  closed bool
}
func (p *syncBuffer) Write(b []byte) (n int,err error) {
  p.Cond.L.Lock()
  defer p.Cond.L.Unlock()
  if p.closed {
    return 0,io.EOF
  }
  n, err = p.buffer.Write(b)
  if n != 0 {
    p.Signal()
  }
  return
}

func (p *syncBuffer) Read(b []byte) (n int,err error) {
  p.Cond.L.Lock()
  defer p.Cond.L.Unlock()
  if p.closed {
    return 0,io.EOF
  }else if p.buffer.Len() == 0 {
    p.Wait()
    if p.closed {
      return 0,io.EOF
    }
    n,err = p.buffer.Read(b)
    if p.buffer.Len() != 0 {
      p.Signal()
    }
  }else{
    n,err = p.buffer.Read(b)
  }
  return
}

func (p *syncBuffer) Close() error {
  p.Cond.L.Lock()
  defer p.Cond.L.Unlock()
  if p.closed {
    return Closed
  }
  p.closed = true
  p.Broadcast()
  return nil
}

func (p *syncBuffer) IsClosed() bool {
  return p.closed
}

func newSyncBuffer() *syncBuffer{
  var mux sync.Mutex
  return &syncBuffer{
    Cond: deadlinecond.NewCond(&mux),
  }
}

type pipe struct{
  *syncBuffer
  io.WriteCloser
  addr net.Addr
}

func (w *pipe) Write(b []byte) (int, error){
  return w.WriteCloser.Write(b)
}

func (w *pipe) LocalAddr() net.Addr {
  return w.addr
}

func (w *pipe) RemoteAddr() net.Addr {
  return w.addr
}
func (w *pipe) SetDeadline(t time.Time) error {
  return w.SetReadDeadline(t)
}
func (w *pipe) SetWriteDeadline(t time.Time) error {
  return nil // writing never times out
}
func (w *pipe) SetReadDeadline(t time.Time) error {
  w.syncBuffer.Cond.SetDeadline(t)
  return nil
}
func (w *pipe) Close() error {
  w.syncBuffer.Close()
  return w.WriteCloser.Close()
}

type PipeListener struct {
  NetAddr net.Addr
  incoming chan chan net.Conn
}

func Listen() *PipeListener {
  fa   := fakeAddr("pipe://")
  return &PipeListener{
    NetAddr: &fa,
    incoming: make(chan chan net.Conn,1),
  }
}

func (p *PipeListener) Addr() net.Addr {
  return p.NetAddr
}

func (p *PipeListener) Accept() (net.Conn, error) {
  if ch,ok := <-p.incoming ; ok {
    bufa := newSyncBuffer()
    bufb := newSyncBuffer()
    fa   := fakeAddr("pipe://")
    a    := &pipe{ bufa, bufb, &fa }
    b    := &pipe{ bufb, bufa, &fa }
    ch <- a
    return b, nil
  }else{
    return nil,Closed
  }
}

func (p *PipeListener) Connect() (net.Conn, error) {
  ch := make(chan net.Conn,1)
  defer func(){ close(ch) }()

  select{
  case p.incoming <- ch :
    if con := <-ch ; con != nil {
      return con, nil
    }else{
      return nil, Closed
    }
  default:
    return nil, Closed
  }
}

func (p *PipeListener) Close() error {
  close(p.incoming)
  return nil
}
