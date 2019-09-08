package beater

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/tmechen/mqttbeat/config"
)

// Mqttbeat configuration.
type Mqttbeat struct {
	done       chan struct{}
	config     config.Config
	client     beat.Client
	mqttClient MQTT.Client
}

func setupMqttClient(bt *Mqttbeat) {
	mqttClientOpt := MQTT.NewClientOptions()
	mqttClientOpt.AddBroker(bt.config.BrokerURL)
	logp.Info("BROKER url: %s", bt.config.BrokerURL)
	mqttClientOpt.SetConnectionLostHandler(bt.reConnectHandler)
	mqttClientOpt.SetOnConnectHandler(bt.subscribeOnConnect)

	if bt.config.BrokerUsername != "" && bt.config.BrokerPassword != "" {
		logp.Info("BROKER username: %s", bt.config.BrokerUsername)
		mqttClientOpt.SetUsername(bt.config.BrokerUsername)
		mqttClientOpt.SetPassword(bt.config.BrokerPassword)
	}
	bt.mqttClient = MQTT.NewClient(mqttClientOpt)
}

func (bt *Mqttbeat) connect(client MQTT.Client) {
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logp.Info("Failed to connect to broker, waiting 5 seconds and retrying")
		time.Sleep(5 * time.Second)
		bt.reConnectHandler(client, token.Error())
		return
	}
	logp.Info("MQTT Client connected: %t", client.IsConnected())
	bt.mqttClient = client
}

func (bt *Mqttbeat) subscribeOnConnect(client MQTT.Client) {
	subscriptions := ParseTopics(bt.config.TopicsSubscribe)

	// Mqtt client - Subscribe to every topic in the config file, and bind with message handler
	if token := bt.mqttClient.SubscribeMultiple(subscriptions, bt.onMessage); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	logp.Info("Subscribed to configured topics")
}

// New creates an instance of mqttbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Mqttbeat{
		done:   make(chan struct{}),
		config: config,
	}
	setupMqttClient(bt)
	bt.connect(bt.mqttClient)
	return bt, nil
}

func (bt *Mqttbeat) onMessage(client MQTT.Client, msg MQTT.Message) {
	logp.Debug("mqttbeat", "MQTT MESSAGE RECEIVED %s", string(msg.Payload()))
	var err error
	event := beat.Event{
		Timestamp: time.Now(),
	}
	event.Fields, err = DecodePayload(msg.Topic(), string(msg.Payload()))
	if err != nil {
		event.Fields = common.MapStr{
			"beat":    common.MapStr{"index": "mqttbeat", "type": "message"},
			"topic":   msg.Topic(),
			"msg_raw": string(msg.Payload()),
			"decoded": false,
		}
	}

	// Finally sending the message to elasticsearch
	bt.client.Publish(event)
}

// DefaultConnectionLostHandler does nothing
func (bt *Mqttbeat) reConnectHandler(client MQTT.Client, reason error) {
	logp.Warn("Connection lost: %s", reason.Error())
	bt.connect(client)
}

// Run is used to start this beater, once configured and connected
func (bt *Mqttbeat) Run(b *beat.Beat) error {
	logp.Info("mqttbeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}
	// The mqtt client is asynchronous, so here we don't have anuthing to do
	for {
		select {
		case <-bt.done:
			return nil
		}
	}
}

// Stop is used to close this beater
func (bt *Mqttbeat) Stop() {
	bt.mqttClient.Disconnect(250)
	bt.client.Close()
	close(bt.done)
}

// ParseTopics will parse the config file and return a map with topic:QoS
func ParseTopics(topics []string) map[string]byte {
	subscriptions := make(map[string]byte)
	for _, value := range topics {
		// Fist, spliting the string topic?qos
		topic, qosStr := strings.Split(value, "?")[0], strings.Split(value, "?")[1]
		// Then, parsing the qos to an int
		qosInt, err := strconv.ParseInt(qosStr, 10, 0)
		if err != nil {
			panic("Error parsing topics")
		}
		// Finally, filling the subscriptions map
		subscriptions[topic] = byte(qosInt)
	}
	return subscriptions
}

func DecodePayload(topic string, payload string) (common.MapStr, error) {
	decodedMsg := make(common.MapStr)
	err := json.Unmarshal([]byte(payload), &decodedMsg)
	if err == nil {
		logp.Debug("mqttbeat", "Payload decoded")
		fields := common.MapStr{
			"beat":    common.MapStr{"index": "mqttbeat", "type": "message"},
			"topic":   topic,
			"decoded": true,
			"msg":     decodedMsg,
			"msg_raw": payload,
		}
		logp.Debug("mqttbeat", "decodedMsg", decodedMsg)
		return fields, err
	}
	return decodedMsg, err
}
