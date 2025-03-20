package amqp

import (
	"encoding/json"
	"mango/pkg/chanrpc"
	"mango/pkg/log"
	"github.com/GoldBaby5511/go-simplejson"
	"github.com/streadway/amqp"
)

const (
	RabbitMqMessageNotifyId string = "RabbitMqMessageNotifyId"
)

type RabbitMQMessage struct {
	Time int64  `json:"timestamp"`
	Body string `json:"body"`
}

var (
	MsgRouter *chanrpc.Server  = nil
	conn      *amqp.Connection = nil
	ch        *amqp.Channel    = nil
)

func NewConsumer(c string) {
	if c == "" {
		return
	}
	var err error
	var jsonConfig *simplejson.Json
	jsonConfig, err = simplejson.NewJson([]byte(c))
	if err != nil {
		log.Warning("database", "RabbitMq配置异常,MqConfig=%v,err=%v", c, err)
		return
	}
	url := jsonConfig.Get("url").MustString("")
	conn, err = amqp.Dial(url)
	if err != nil {
		log.Error("", "异常,MQ连接失败,url=%v,err=%v", url, err)
		return
	}
	ch, err = conn.Channel()
	if err != nil {
		log.Error("", "异常,连接channel失败,err=%v", err)
		return
	}
	queueName := jsonConfig.Get("queueName").MustString("")
	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Error("", "异常,创建队列失败,name=%v,err=%v", queueName, err)
		return
	}
	msg, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Error("", "异常,注册消费者失败,name=%v,err=%v", q.Name, err)
		return
	}
	go func() {
		for d := range msg {
			if MsgRouter != nil {
				log.Debug("", "Received a message: %s", d.Body)
				var m RabbitMQMessage
				if err := json.Unmarshal(d.Body, &m); err == nil {
					MsgRouter.Go(RabbitMqMessageNotifyId, m)
				} else {
					log.Warning("", "消息序列化失败?,body=%v", d.Body)
				}
			} else {
				log.Warning("", "没有消息路由?,body=%v", d.Body)
			}
		}
	}()

	log.Info("", "MQ连接成功,url=%v,queue=%v", url, queueName)
}

func Close() {
	if conn != nil {
		conn.Close()
	}
	if ch != nil {
		ch.Close()
	}
}
