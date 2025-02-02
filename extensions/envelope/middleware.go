package envelope

import (
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/hyperledger-labs/cckit/router"
	"go.uber.org/zap"
)

const (
	// argument indexes
	methodNamePos = iota
	payloadPos
	envelopePos

	nonceObjectType = "nonce"
	invokeType      = "invoke"

	TimeLayout = "2006-01-02T15:04:05.000Z"

	PubKey string = "envelopePubkey" // router context key
)

// middleware for checking signature that is got in envelop
func Verify() router.MiddlewareFunc {
	return func(next router.HandlerFunc, pos ...int) router.HandlerFunc {
		return func(c router.Context) (interface{}, error) {
			if c.Handler().Type == invokeType {
				iArgs := c.GetArgs()
				if len(iArgs) == 2 {
					c.Logger().Sugar().Error(ErrSignatureNotFound)
					return nil, ErrSignatureNotFound
				} else {
					var (
						e   *Envelope
						err error
					)
					if e, err = verifyEnvelope(c, iArgs[methodNamePos], iArgs[payloadPos], iArgs[envelopePos]); err != nil {
						return nil, err
					}
					// store corect pubkey in context
					c.SetParam(PubKey, e.PublicKey)
				}
			}
			return next(c)
		}
	}
}

func verifyEnvelope(c router.Context, method, payload, envlp []byte) (*Envelope, error) {
	// parse json envelope format (json is original format for envelope from frontend)
	data, err := c.Serializer().FromBytesTo(envlp, &Envelope{})
	if err != nil {
		c.Logger().Error(`convert from bytes failed:`, zap.Error(err))
		return nil, err
	}
	envelope := data.(*Envelope)
	if envelope.Deadline.AsTime().Unix() != 0 {
		if envelope.Deadline.AsTime().Unix() < time.Now().Unix() {
			c.Logger().Sugar().Error(ErrDeadlineExpired)
			return nil, ErrDeadlineExpired
		}
	}

	// check method and channel names because envelope can only be used once for channel+chaincode+method combination
	if string(method) != envelope.Method {
		c.Logger().Sugar().Error(ErrInvalidMethod)
		return nil, ErrInvalidMethod
	}
	if string(c.Stub().GetChannelID()) != envelope.Channel {
		c.Logger().Sugar().Error(ErrInvalidChannel)
		return nil, ErrInvalidChannel
	}

	// replay attack check
	txHash := txNonceKey(payload, envelope.Nonce, envelope.Channel, envelope.Chaincode, envelope.Method, envelope.PublicKey)
	key, err := c.Stub().CreateCompositeKey(nonceObjectType, []string{txHash})
	if err != nil {
		return nil, err
	}
	bb, err := c.Stub().GetState(key)
	if bb == nil && err == nil {
		if err := c.Stub().PutState(key, []byte{'0'}); err != nil {
			return nil, err
		}
		// convert public key and sig from base58
		pubkey := base58.Decode(envelope.PublicKey)
		sig := base58.Decode(envelope.Signature)
		// convert deadline to frontend format
		var deadline string
		if envelope.Deadline != nil {
			deadline = envelope.Deadline.AsTime().Format(TimeLayout)
		}
		if err := CheckSig(payload, envelope.Nonce, envelope.Channel, envelope.Chaincode, envelope.Method, deadline, pubkey, sig); err != nil {
			c.Logger().Sugar().Error(ErrCheckSignatureFailed)
			return nil, ErrCheckSignatureFailed
		}
	} else {
		c.Logger().Sugar().Error(ErrTxAlreadyExecuted)
		return nil, ErrTxAlreadyExecuted
	}
	return envelope, nil
}

func txNonceKey(payload []byte, nonce, channel, chaincode, method, pubKey string) string {
	bb := append(payload, pubKey...)
	bb = append(bb, nonce...)
	bb = append(bb, channel...)
	bb = append(bb, chaincode...)
	bb = append(bb, method...)
	hashed := sha256.Sum256(bb)
	return base64.StdEncoding.EncodeToString(hashed[:])
}
