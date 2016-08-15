package main

import (
	"io"
	"log"
	"os"
)

type LeakyBuffer struct {
	in       io.Reader
	out      io.Writer
	log      *log.Logger
	bufSize  int
	stop     chan struct{}
	write    chan []byte
	submit   chan []byte
	unsubmit chan []byte
	recycle  chan []byte
	errno    int
}

func main() {
	os.Exit(NewLeakyBuffer(os.Stdin, os.Stdout, 1<<21).Run())
}

func NewLeakyBuffer(in io.Reader, out io.Writer, bufSize int) *LeakyBuffer {
	return &LeakyBuffer{
		in:       in,
		out:      out,
		log:      log.New(os.Stderr, "", 0),
		bufSize:  bufSize,
		stop:     make(chan struct{}),
		write:    make(chan []byte),
		submit:   make(chan []byte),
		unsubmit: make(chan []byte),
		recycle:  make(chan []byte, 2),
	}
}

func (lb *LeakyBuffer) Run() int {
	go lb.reader()
	go lb.submitter()
	go lb.writer()
	<-lb.stop
	return lb.errno
}

func (lb *LeakyBuffer) reader() {
	defer close(lb.write)
	defer close(lb.submit)

	readbuf := make([]byte, 1<<16)
	buf := make([]byte, 0, lb.bufSize)
	lbuf := 0
	lb.recycle <- make([]byte, 0, lb.bufSize)

	for {
		select {
		case <-lb.stop:
			return
		default:
			bytes, err := lb.in.Read(readbuf)

			if lbuf > 0 {
				buf = <-lb.unsubmit
				lbuf = len(buf)
			}

			if lb.bufSize-lbuf >= bytes {
				buf = append(buf, readbuf[0:bytes]...)
				lbuf = len(buf)
			} else if bytes > 0 {
				// we lose the contents of readbuf
				lb.log.Printf("warn: dropped %d bytes", bytes)
			}

			if err != nil {
				if err != io.EOF {
					lb.log.Printf("fatal: reading: %s", err)
					lb.errno = 2
				}
				lb.write <- buf
				return
			}

			if lbuf > 0 {
				lb.submit <- buf
			}
		}
	}
}

func (lb *LeakyBuffer) submitter() {
	defer close(lb.unsubmit)

	for {
		select {
		case <-lb.stop:
			return
		case buf, ok := <-lb.submit:
			if !ok {
				return
			}
			select {
			case <-lb.stop:
				lb.unsubmit <- buf
				return
			case lb.unsubmit <- buf:
			case lb.write <- buf:
				lb.unsubmit <- <-lb.recycle
			}
		}
	}
}

func (lb *LeakyBuffer) writer() {
	defer close(lb.stop)

	for {
		select {
		case <-lb.stop:
			return
		case buf, ok := <-lb.write:
			if !ok {
				return
			}
			_, err := lb.out.Write(buf)
			lb.recycle <- buf[:0]
			if err != nil {
				lb.log.Printf("fatal: writing: %s", err)
				lb.errno = 1
				return
			}
		}
	}
}
