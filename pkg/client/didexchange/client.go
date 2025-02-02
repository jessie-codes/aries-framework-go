/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package didexchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/common/service"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
	"github.com/hyperledger/aries-framework-go/pkg/wallet"
)

const (
	// ConnectionID connection id is created to retriever connection record from db
	ConnectionID = didexchange.ConnectionID
	// InvitationID invitation id is created in invitation request
	InvitationID = didexchange.InvitationID
)

// ErrConnectionNotFound is returned when connection not found
var ErrConnectionNotFound = errors.New("connection not found")

// provider contains dependencies for the DID exchange protocol and is typically created by using aries.Context()
type provider interface {
	Service(id string) (interface{}, error)
	CryptoWallet() wallet.Crypto
	InboundTransportEndpoint() string
	StorageProvider() storage.Provider
}

// Client enable access to didexchange api
// TODO add support for Accept Exchange Request & Accept Invitation
//  using events & callback (#198 & #238)
type Client struct {
	didexchangeSvc           service.DIDComm
	wallet                   wallet.Crypto
	inboundTransportEndpoint string
	actionCh                 chan service.DIDCommAction
	msgCh                    chan service.StateMsg
	actionEvent              chan<- service.DIDCommAction
	actionEventlock          sync.RWMutex
	msgEvents                []chan<- service.StateMsg
	msgEventsLock            sync.RWMutex
	connectionStore          *didexchange.ConnectionRecorder
}

// New return new instance of didexchange client
func New(ctx provider) (*Client, error) {
	svc, err := ctx.Service(didexchange.DIDExchange)
	if err != nil {
		return nil, err
	}

	didexchangeSvc, ok := svc.(service.DIDComm)
	if !ok {
		return nil, errors.New("cast service to DIDExchange Service failed")
	}

	store, err := ctx.StorageProvider().OpenStore(didexchange.DIDExchange)
	if err != nil {
		return nil, err
	}

	c := &Client{
		didexchangeSvc:           didexchangeSvc,
		wallet:                   ctx.CryptoWallet(),
		inboundTransportEndpoint: ctx.InboundTransportEndpoint(),
		// TODO channel size - https://github.com/hyperledger/aries-framework-go/issues/246
		actionCh:        make(chan service.DIDCommAction, 10),
		msgCh:           make(chan service.StateMsg, 10),
		connectionStore: didexchange.NewConnectionRecorder(store),
	}

	// start listening for action/message events
	err = c.startServiceEventListener()
	if err != nil {
		return nil, fmt.Errorf("service event listener startup failed: %w", err)
	}

	return c, nil
}

// CreateInvitation create invitation
func (c *Client) CreateInvitation(label string) (*didexchange.Invitation, error) {
	verKey, err := c.wallet.CreateEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed CreateSigningKey: %w", err)
	}

	invitation := &didexchange.Invitation{
		ID:              uuid.New().String(),
		Label:           label,
		RecipientKeys:   []string{verKey},
		ServiceEndpoint: c.inboundTransportEndpoint,
		Type:            didexchange.ConnectionInvite,
	}

	err = c.connectionStore.SaveInvitation(verKey, invitation)
	if err != nil {
		return nil, fmt.Errorf("failed to save invitation: %w", err)
	}

	return invitation, nil
}

// HandleInvitation handle incoming invitation
func (c *Client) HandleInvitation(invitation *didexchange.Invitation) error {
	payload, err := json.Marshal(invitation)
	if err != nil {
		return fmt.Errorf("failed marshal invitation: %w", err)
	}
	if err = c.didexchangeSvc.Handle(&service.DIDCommMsg{Type: invitation.Type, Payload: payload}); err != nil {
		return fmt.Errorf("failed from didexchange service handle: %w", err)
	}
	return nil
}

// QueryConnections queries connections matching given parameters
func (c *Client) QueryConnections(request *QueryConnectionsParams) ([]*ConnectionResult, error) {
	// TODO sample response, to be implemented as part of #226
	return []*ConnectionResult{
		{didexchange.ConnectionRecord{ConnectionID: uuid.New().String()}},
		{didexchange.ConnectionRecord{ConnectionID: uuid.New().String()}},
	}, nil
}

// GetConnection fetches single connection record for given id
func (c *Client) GetConnection(connectionID string) (*ConnectionResult, error) {
	conn, err := c.connectionStore.GetConnection(connectionID)
	if err != nil {
		if errors.Is(err, storage.ErrDataNotFound) {
			return nil, ErrConnectionNotFound
		}
		return nil, fmt.Errorf("cannot fetch state from store: connectionid=%s err=%s", connectionID, err)
	}
	return &ConnectionResult{
		didexchange.ConnectionRecord{ConnectionID: connectionID, State: conn.State},
	}, nil
}

// RemoveConnection removes connection record for given id
func (c *Client) RemoveConnection(id string) error {
	// TODO sample response, to be implemented as part of #226
	return nil
}

// startServiceEventListener listens to action and message events from DID Exchange service.
func (c *Client) startServiceEventListener() error {
	err := c.didexchangeSvc.RegisterActionEvent(c.actionCh)
	if err != nil {
		return fmt.Errorf("didexchange action event registration failed: %w", err)
	}

	// register the message event channel
	err = c.didexchangeSvc.RegisterMsgEvent(c.msgCh)
	if err != nil {
		return fmt.Errorf("didexchange message event registration failed: %w", err)
	}

	// listen for action event and message events
	go func() {
		for {
			select {
			case e := <-c.actionCh:
				// assigned to var as lint fails with : Using a reference for the variable on range scope (scopelint)
				msg := e
				c.handleActionEvent(&msg)
			case e := <-c.msgCh:
				// assigned to var as lint fails with : Using a reference for the variable on range scope (scopelint)
				msg := e
				c.handleMessageEvent(&msg)
			}
		}
	}()

	return nil
}

// RegisterActionEvent on DID Exchange protocol messages. The events are triggered for incoming exchangeRequest,
// exchangeResponse and exchangeAck message types. The consumer need to invoke the callback to resume processing.
// Only one channel can be registered for the action events. The function will throw error if a channel is already
// registered. The AutoExecuteActionEvent() function can be used to automatically trigger callback function for the
// event.
func (c *Client) RegisterActionEvent(ch chan<- service.DIDCommAction) error {
	c.actionEventlock.Lock()
	defer c.actionEventlock.Unlock()

	if c.actionEvent != nil {
		return errors.New("channel is already registered for the action event")
	}

	c.actionEvent = ch

	return nil
}

// UnregisterActionEvent on DID Exchange protocol messages. Refer RegisterActionEvent().
func (c *Client) UnregisterActionEvent(ch chan<- service.DIDCommAction) error {
	c.actionEventlock.Lock()
	defer c.actionEventlock.Unlock()

	if c.actionEvent != ch {
		return errors.New("invalid channel passed to unregister the action event")
	}

	c.actionEvent = nil

	return nil
}

// RegisterMsgEvent on DID Exchange protocol messages. The message events are triggered for state transitions. Client
// will not expect any callback on these events unlike Action events.
func (c *Client) RegisterMsgEvent(ch chan<- service.StateMsg) error {
	c.msgEventsLock.Lock()
	c.msgEvents = append(c.msgEvents, ch)
	c.msgEventsLock.Unlock()

	return nil
}

// UnregisterMsgEvent on DID Exchange protocol messages.
func (c *Client) UnregisterMsgEvent(ch chan<- service.StateMsg) error {
	c.msgEventsLock.Lock()
	for i := 0; i < len(c.msgEvents); i++ {
		if c.msgEvents[i] == ch {
			c.msgEvents = append(c.msgEvents[:i], c.msgEvents[i+1:]...)
			i--
		}
	}
	c.msgEventsLock.Unlock()

	return nil
}

func (c *Client) handleActionEvent(msg *service.DIDCommAction) {
	c.actionEventlock.RLock()
	aEvent := c.actionEvent
	c.actionEventlock.RLock()

	aEvent <- *msg
}

func (c *Client) handleMessageEvent(msg *service.StateMsg) {
	c.msgEventsLock.RLock()
	statusEvents := c.msgEvents
	c.msgEventsLock.RUnlock()

	for _, handler := range statusEvents {
		handler <- *msg
	}
}
