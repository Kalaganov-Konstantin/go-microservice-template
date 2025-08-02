package http

import (
	"context"
	"fmt"
	"microservice/internal/config"
	"microservice/internal/platform/logger"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	logger logger.Logger
}

func (s *ServerTestSuite) SetupTest() {
	s.logger = logger.NewNop()
}

func (s *ServerTestSuite) TestNewServer() {
	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
	}

	handler := http.NewServeMux()

	server := NewServer(cfg, s.logger, handler)

	s.Assert().NotNil(server)
	s.Assert().NotNil(server.server)
	s.Assert().Equal(s.logger, server.logger)
	s.Assert().Equal("localhost:8080", server.server.Addr)
	s.Assert().Equal(handler, server.server.Handler)
	s.Assert().Equal(30*time.Second, server.server.ReadTimeout)
	s.Assert().Equal(30*time.Second, server.server.WriteTimeout)
	s.Assert().Equal(120*time.Second, server.server.IdleTimeout)
}

func (s *ServerTestSuite) TestNewServer_DefaultHost() {
	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "",
			Port:         9000,
			ReadTimeout:  10,
			WriteTimeout: 15,
			IdleTimeout:  60,
		},
	}

	handler := http.NewServeMux()

	server := NewServer(cfg, s.logger, handler)

	s.Assert().NotNil(server)
	s.Assert().Equal(":9000", server.server.Addr)
	s.Assert().Equal(10*time.Second, server.server.ReadTimeout)
	s.Assert().Equal(15*time.Second, server.server.WriteTimeout)
	s.Assert().Equal(60*time.Second, server.server.IdleTimeout)
}

func (s *ServerTestSuite) TestServer_Start_Success() {
	listener, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	port := listener.Addr().(*net.TCPAddr).Port
	s.Require().NoError(listener.Close())

	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "localhost",
			Port:         port,
			ReadTimeout:  5,
			WriteTimeout: 5,
			IdleTimeout:  10,
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	})

	server := NewServer(cfg, s.logger, handler)

	ctx := context.Background()
	err = server.Start(ctx)
	s.Assert().NoError(err)

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	s.Assert().NoError(err)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NoError(resp.Body.Close())

	err = server.Stop(ctx)
	s.Assert().NoError(err)
}

func (s *ServerTestSuite) TestServer_Start_InvalidPort() {
	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host: "127.0.0.1",
			Port: 99999,
		},
	}

	handler := http.NewServeMux()
	server := NewServer(cfg, s.logger, handler)

	ctx := context.Background()
	err := server.Start(ctx)
	s.Assert().Error(err)
}

func (s *ServerTestSuite) TestServer_Stop_Success() {
	listener, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	port := listener.Addr().(*net.TCPAddr).Port
	s.Require().NoError(listener.Close())

	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host: "localhost",
			Port: port,
		},
	}

	handler := http.NewServeMux()
	server := NewServer(cfg, s.logger, handler)

	ctx := context.Background()

	err = server.Start(ctx)
	s.Assert().NoError(err)

	time.Sleep(100 * time.Millisecond)

	err = server.Stop(ctx)
	s.Assert().NoError(err)

	time.Sleep(100 * time.Millisecond)
	_, err = http.Get(fmt.Sprintf("http://localhost:%d/", port))
	s.Assert().Error(err)
}

func (s *ServerTestSuite) TestServer_Stop_NilServer() {
	server := &Server{
		server: nil,
		logger: s.logger,
	}

	ctx := context.Background()
	err := server.Stop(ctx)
	s.Assert().NoError(err)
}

func (s *ServerTestSuite) TestServer_Stop_WithTimeout() {
	listener, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	port := listener.Addr().(*net.TCPAddr).Port
	s.Require().NoError(listener.Close())

	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host: "localhost",
			Port: port,
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(50 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
	})

	server := NewServer(cfg, s.logger, handler)

	ctx := context.Background()

	err = server.Start(ctx)
	s.Assert().NoError(err)

	time.Sleep(100 * time.Millisecond)

	err = server.Stop(ctx)
	s.Assert().NoError(err)
}

func (s *ServerTestSuite) TestServer_StartStop_Multiple() {
	listener, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	port := listener.Addr().(*net.TCPAddr).Port
	s.Require().NoError(listener.Close())

	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host: "localhost",
			Port: port,
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewServer(cfg, s.logger, handler)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		err = server.Start(ctx)
		s.Assert().NoError(err)

		time.Sleep(50 * time.Millisecond)

		err = server.Stop(ctx)
		s.Assert().NoError(err)

		time.Sleep(50 * time.Millisecond)
	}
}

func (s *ServerTestSuite) TestServer_Integration() {
	listener, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	port := listener.Addr().(*net.TCPAddr).Port
	s.Require().NoError(listener.Close())

	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "localhost",
			Port:         port,
			ReadTimeout:  1,
			WriteTimeout: 1,
			IdleTimeout:  2,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("root"))
	})
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"test"}`))
	})

	server := NewServer(cfg, s.logger, mux)
	ctx := context.Background()

	err = server.Start(ctx)
	s.Assert().NoError(err)

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	s.Assert().NoError(err)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Require().NoError(resp.Body.Close())

	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/api/test", port))
	s.Assert().NoError(err)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("application/json", resp.Header.Get("Content-Type"))
	s.Require().NoError(resp.Body.Close())

	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/nonexistent", port))
	s.Assert().NoError(err)
	s.Assert().Equal(http.StatusNotFound, resp.StatusCode)
	s.Require().NoError(resp.Body.Close())

	err = server.Stop(ctx)
	s.Assert().NoError(err)
}

func (s *ServerTestSuite) TestServer_Performance() {
	listener, err := net.Listen("tcp", ":0")
	s.Require().NoError(err)
	port := listener.Addr().(*net.TCPAddr).Port
	s.Require().NoError(listener.Close())

	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host: "localhost",
			Port: port,
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := NewServer(cfg, s.logger, handler)
	ctx := context.Background()

	err = server.Start(ctx)
	s.Assert().NoError(err)

	time.Sleep(100 * time.Millisecond)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
			s.Assert().NoError(err)
			s.Assert().Equal(http.StatusOK, resp.StatusCode)
			s.Require().NoError(resp.Body.Close())
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	err = server.Stop(ctx)
	s.Assert().NoError(err)
}

func BenchmarkNewServer(b *testing.B) {
	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
	}
	handler := http.NewServeMux()
	log := logger.NewNop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server := NewServer(cfg, log, handler)
		_ = server
	}
}

func BenchmarkServer_StartStop(b *testing.B) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		b.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	cfg := &config.HttpConfig{
		Server: config.HttpServerConfig{
			Host: "localhost",
			Port: port,
		},
	}
	handler := http.NewServeMux()
	log := logger.NewNop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server := NewServer(cfg, log, handler)

		err := server.Start(ctx)
		if err != nil {
			b.Fatal(err)
		}

		time.Sleep(10 * time.Millisecond)

		err = server.Stop(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
