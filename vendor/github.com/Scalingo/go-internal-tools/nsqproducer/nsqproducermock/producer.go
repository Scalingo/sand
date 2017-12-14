package nsqproducermock

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/Scalingo/go-internal-tools/nsqproducer"

	"gopkg.in/errgo.v1"
)

type ProducerMock struct {
	sync.Mutex
	messages map[string][]nsqproducer.NsqMessageSerialize
}

func New() *ProducerMock {
	return &ProducerMock{
		messages: make(map[string][]nsqproducer.NsqMessageSerialize),
	}
}

func (producer *ProducerMock) CountTopic(topic string) int {
	producer.Lock()
	defer producer.Unlock()
	if messages, ok := producer.messages[topic]; !ok {
		return 0
	} else {
		return len(messages)
	}
}

func (producer *ProducerMock) Messages(topic string) []nsqproducer.NsqMessageSerialize {
	producer.Lock()
	defer producer.Unlock()
	if messages, ok := producer.messages[topic]; !ok {
		return []nsqproducer.NsqMessageSerialize{}
	} else {
		return messages
	}
}

func (producer *ProducerMock) UnmarshallLastMessage(topic string, data interface{}) error {
	producer.Lock()
	defer producer.Unlock()
	if messages, ok := producer.messages[topic]; !ok {
		return errgo.Newf("no message in topic %s", topic)
	} else {
		if reflect.TypeOf(data).Kind() == reflect.Ptr {
			reflect.ValueOf(data).Elem().Set(reflect.ValueOf(messages[len(messages)-1].Payload))
		} else {
			return errgo.New("data should be a pointer")
		}
	}
	return nil
}

func (producer *ProducerMock) UnmarshalAllMessages(topic string, data interface{}) error {
	producer.Lock()
	defer producer.Unlock()
	if messages, ok := producer.messages[topic]; !ok {
		return errgo.Newf("no message in topic %s", topic)
	} else {
		if reflect.TypeOf(data).Kind() == reflect.Ptr {
			reflect.ValueOf(data).Elem().Set(reflect.ValueOf(messages))
		} else {
			return errgo.New("data should be a pointer")
		}
	}
	return nil
}

func (producer *ProducerMock) Publish(ctx context.Context, topic string, message nsqproducer.NsqMessageSerialize) error {
	producer.Lock()
	producer.messages[topic] = append(producer.messages[topic], message)
	producer.Unlock()
	return nil
}

func (producer *ProducerMock) DeferredPublish(ctx context.Context, topic string, delay int64, message nsqproducer.NsqMessageSerialize) error {
	message.At = time.Now().Add(time.Duration(delay) * time.Second).Unix()
	producer.Lock()
	producer.messages[topic] = append(producer.messages[topic], message)
	producer.Unlock()
	return nil
}
