package env

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"

	"github.com/cedws/iapc/iap"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
	"github.com/coreeng/corectl/pkg/logger"
	"go.uber.org/zap"
)

// Listen starts a proxy server that listens on the given address and port.
func Listen(streams userio.IOStreams, opts EnvConnectOpts, ctx context.Context, listen string, dialOpts []iap.DialOption, execute func() error) {
	wizardH := streams.CurrentHandler

	var listener net.Listener
	var err error

	if IsConnectStartup(opts) { // Common code for foreground and background
		wizardH.SetTask("Testing IAP connection", "IAP connection succeeded")
		if err := testConn(ctx, dialOpts); err != nil {
			err = fmt.Errorf("failed to test connection: %w", err)
			wizardH.Abort(err.Error())
			logger.Fatal().Msg(err.Error())
		}
		wizardH.SetCurrentTaskCompleted()
		listener, err = net.Listen("tcp", listen)

		wizardH.SetTask(fmt.Sprintf("Binding to %s", listen), "")

		if err != nil {
			wizardH.SetCurrentTaskCompletedTitleWithStatus(
				fmt.Sprintf("failed to bind to port: %s", err), wizard.TaskStatusError)
			logger.Fatal().With(zap.Error(err)).Msgf("failed to bind to %s", listen)
			return
		}
		wizardH.SetCurrentTaskCompletedTitle(fmt.Sprintf("Bound to %s", listen))
		logger.Info().Msgf("listening: %+v", listener)
		if !opts.Background {
			WritePidFile(opts.Environment.Environment, os.Getpid())
		}
	}

	if IsConnectParent(opts) {
		// background parent specific logic
		file, err := listener.(*net.TCPListener).File()
		if err != nil {
			logger.Fatal().With(zap.Error(err)).Msg("failed to get file descriptor")
		}
		if err := listener.Close(); err != nil {
			logger.Fatal().With(zap.Error(err)).Msg("failed to close listener")
		}

		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(), SetBackgroundEnv())
		cmd.ExtraFiles = []*os.File{file}
		err = cmd.Start()
		if err != nil {
			logger.Fatal().With(zap.Error(err)).Msg("failed to start background process")
		}
		WritePidFile(opts.Environment.Environment, cmd.Process.Pid)
		wizardH.SetTask("", fmt.Sprintf("Process started for %s in background with PID %d",
			opts.Environment.Environment, cmd.Process.Pid),
		)
		return
	}
	if IsConnectChild(opts) {
		// background child specific logic
		fileListener := os.NewFile(uintptr(3), "listener")

		listener, err = net.FileListener(fileListener)
		if err != nil {
			logger.Fatal().With(zap.Error(err)).Msg("failed to create listener from file descriptor")
		}
		fileListener.Close()
	}

	executionFinished := make(chan error)
	go func() {
		if execute != nil {
			err := execute()
			listener.Close()
			executionFinished <- err
		}
	}()

	for {
		select {
		case <-executionFinished:
			logger.Warn().Msg("Execution finished, no longer accepting new connections.")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					logger.Warn().Msg("Listener closed, stopping new connections.")
					err := <-executionFinished
					if err != nil {
						wizardH.Abort(err.Error())
						logger.Fatal().With(zap.Error(err)).Msg("Execution failed")
					}
					wizardH.Warn("Tunnel closed")
					return
				}
				logger.Fatal().With(zap.Error(err)).Msg("failed to accept connection")
			}

			go handleClient(ctx, wizardH, dialOpts, conn)
		}
	}
}

func testConn(ctx context.Context, opts []iap.DialOption) error {
	tun, err := iap.Dial(ctx, opts...)
	if tun != nil {
		defer tun.Close()
	}
	return err
}

func handleClient(ctx context.Context, wizard wizard.Handler, opts []iap.DialOption, conn net.Conn) {
	logger.Debug().Msgf("connected: client %s", conn.RemoteAddr())

	tun, err := iap.Dial(ctx, opts...)
	if err != nil {
		wizard.Error(fmt.Sprintf("Failed to connect to IAP for client: %s", conn.RemoteAddr()))
		logger.Error().With(zap.Error(err)).Msgf("failed to dial IAP")
		return
	}
	defer tun.Close()

	logger.Debug().Msgf("iap dialed: client %s | %s -> %s (local)", conn.RemoteAddr(), tun.RemoteAddr(), tun.LocalAddr())

	go func() {
		if _, err := io.Copy(conn, tun); err != nil {
			logger.Debug().With(zap.Error(err)).Msg("failed to transfer data")
		}
	}()
	if _, err := io.Copy(tun, conn); err != nil {
		logger.Debug().With(zap.Error(err)).Msg("")
	}

	logger.Debug().Msgf("disconnected: client %s | sentbytes %d | recvbytes %d", conn.RemoteAddr(), tun.Sent(), tun.Received())
}
