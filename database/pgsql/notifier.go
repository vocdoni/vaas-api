package pgsql

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"
	"go.vocdoni.io/api/config"
	"go.vocdoni.io/api/urlapi"
	"go.vocdoni.io/dvote/log"
)

// notifier encapsulates the state of the listener connection.
type notifier struct {
	listener *pq.Listener
	failed   chan error
}

func NewNotifier(dbc *config.DB, channelName string) (*notifier, error) {
	notifier := &notifier{failed: make(chan error, 2)}
	listener := pq.NewListener(fmt.Sprintf("host=%s port=%d user=%s password=%s"+
		" dbname=%s sslmode=%s client_encoding=%s",
		dbc.Host, dbc.Port, dbc.User, dbc.Password, dbc.Dbname,
		dbc.Sslmode, "UTF8"), 2*time.Second, time.Minute, notifier.logListener)
	if err := listener.Listen(channelName); err != nil {
		listener.Close()
		log.Errorf("could not start auth token listener: %v", err)
		return nil, err
	}
	notifier.listener = listener
	return notifier, nil
}

// fetch is the main loop of the notifier to receive data from
// the database in JSON-FORMAT and send it down the send channel.
func (n *notifier) FetchNewTokens(u *urlapi.URLAPI) {
	for {
		select {
		case e := <-n.listener.Notify:
			if e == nil {
				continue
			}
			delete, token := parseOperation(e.Extra)
			if !delete {
				u.RegisterToken(token, urlapi.INTEGRATOR_MAX_REQUESTS)
			} else {
				u.RevokeToken(token)
			}
			log.Debug("pgsql notified: ", e.Extra)
		case err := <-n.failed:
			log.Error(err)
		case <-time.After(time.Minute):
			go func() {
				err := n.listener.Ping()
				if err != nil {
					log.Error(err)
				}
			}()
		}
	}
}

func parseOperation(op string) (delete bool, token string) {
	if strings.Contains(op, "DELETE") {
		delete = true
	}
	m := regexp.MustCompile(`KEY\s?=?\s?(.*)`)
	token = m.FindStringSubmatch(op)[1]
	return delete, token
}

func (n *notifier) logListener(event pq.ListenerEventType, err error) {
	if err != nil {
		log.Errorf("pgsql listener error: %s\n", err)
	}
	if event == pq.ListenerEventConnectionAttemptFailed {
		n.failed <- err
	}
}
