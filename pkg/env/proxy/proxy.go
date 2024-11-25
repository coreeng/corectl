// Adapted from https://github.com/cedws/iapc/blob/master/internal/proxy/proxy.go
package proxy

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/cedws/iapc/iap"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/cmdutil/userio/wizard"
	"github.com/phuslu/log"
)

// Listen starts a proxy server that listens on the given address and port.
func Listen(streams userio.IOStreams, ctx context.Context, listen string, opts []iap.DialOption, execute func() error) {
	wizardH := streams.CurrentHandler
	wizardH.SetTask("Testing IAP connection", "IAP connection succeeded")
	if err := testConn(ctx, opts); err != nil {
		err = fmt.Errorf("failed to test connection: %w", err)
		wizardH.Abort(err.Error())
		log.Fatal().Msg(err.Error())
	}

	listener, err := net.Listen("tcp", listen)
	wizardH.SetTask(
		fmt.Sprintf("Binding to %s", listen),
		fmt.Sprintf("Bound to %s", listen),
	)
	if err != nil {
		wizardH.Abort(fmt.Errorf("failed to test connection: %w", err).Error())
		log.Fatal().Err(err).Msgf("failed to bind to %s", listen)
	}

	wizardH.SetCurrentTaskCompleted()
	log.Info().Msgf("listening: %+v", listener)

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
			log.Info().Msg("Execution finished, no longer accepting new connections.")
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					log.Info().Msg("Listener closed, stopping new connections.")
					err := <-executionFinished
					if err != nil {
						wizardH.Abort(err.Error())
						log.Fatal().Err(err).Msg("Execution failed")
					}
					wizardH.Info("Tunnel closed")
					return
				}
				log.Fatal().Err(err).Msg("failed to accept connection")
			}

			go handleClient(ctx, wizardH, opts, conn)
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
	log.Debug().Msgf("connected: client %s", conn.RemoteAddr())

	tun, err := iap.Dial(ctx, opts...)
	if err != nil {
		wizard.Error(fmt.Sprintf("Failed to connect to IAP for client: %s", conn.RemoteAddr()))
		log.Error().Err(err).Msgf("failed to dial IAP")
		return
	}
	defer tun.Close()

	log.Debug().Msgf("iap dialed: client %s | %s -> %s (local)", conn.RemoteAddr(), tun.RemoteAddr(), tun.LocalAddr())

	go func() {
		if _, err := io.Copy(conn, tun); err != nil {
			log.Debug().Err(err).Msg("failed to transfer data")
		}
	}()
	if _, err := io.Copy(tun, conn); err != nil {
		log.Debug().Err(err).Msg("")
	}

	log.Debug().Msgf("disconnected: client %s | sentbytes %d | recvbytes %d", conn.RemoteAddr(), tun.Sent(), tun.Received())
}
