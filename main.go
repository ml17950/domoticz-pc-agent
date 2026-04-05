package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	ini "gopkg.in/ini.v1"
)

type Config struct {
	MQTT struct {
		BrokerAddress string `ini:"broker_address"`
		Port          string `ini:"port"`
		Username      string `ini:"username"`
		Password      string `ini:"password"`
	} `ini:"mqtt"`
	Domoticz struct {
		Idx  string `ini:"idx"`
		Type string `ini:"type"`
	} `ini:"domoticz"`
}

// connectHandler is called when the connection to the MQTT broker is successful.
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Successfully connected to the MQTT broker!")
}

// connectLostHandler is called when the connection to the MQTT broker is unexpectedly lost.
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v\n", err)
	// systemd will restart the program if it crashes.
}

// messageHandler is called when a message is received on a subscribed topic.
var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Message received on topic: %s\n", msg.Topic())
	fmt.Printf("Payload: %s\n", msg.Payload())
	// You could add more logic here, e.g., forwarding the message to Domoticz
	// based on the Idx and Type read from the INI file.
}

func main() {
	// Load configuration file
	cfgFile := "domoticz-pc-agent.ini"

	// 1. Load the INI file into an ini.File object
	file, err := ini.Load(cfgFile)
	if err != nil {
		fmt.Printf("Error loading configuration file '%s': %v\n", cfgFile, err)
		os.Exit(1)
	}

	// 2. Map the loaded configuration to the struct
	var cfg Config
	err = file.MapTo(&cfg)
	if err != nil {
		fmt.Printf("Error mapping configuration to struct: %v\n", err)
		os.Exit(1)
	}

	// Check if essential configuration is present
	if cfg.MQTT.BrokerAddress == "" || cfg.MQTT.Port == "" {
		fmt.Println("Error: MQTT Broker address and port must be set in the INI file.")
		os.Exit(1)
	}
	// Output of the read configuration (optional, for debugging)
	fmt.Printf("MQTT Broker: %s:%s\n", cfg.MQTT.BrokerAddress, cfg.MQTT.Port)
	if cfg.MQTT.Username != "" {
		fmt.Printf("MQTT User: %s\n", cfg.MQTT.Username)
	}
	if cfg.Domoticz.Idx != "" {
		fmt.Printf("Domoticz Index: %s\n", cfg.Domoticz.Idx)
	}
	if cfg.Domoticz.Type != "" {
		fmt.Printf("Domoticz Type: %s\n", cfg.Domoticz.Type)
	}

	// Create MQTT client configuration from INI file
	brokerURL := fmt.Sprintf("tcp://%s:%s", cfg.MQTT.BrokerAddress, cfg.MQTT.Port)
	clientID := "domoticz-pc-agent"
	topic := "domoticz" // Keep or read from INI if desired

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(messageHandler) // Process messages for subscriptions
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	// Only set username and password if they are present in the INI file
	if cfg.MQTT.Username != "" {
		opts.SetUsername(cfg.MQTT.Username)
		if cfg.MQTT.Password != "" {
			opts.SetPassword(cfg.MQTT.Password)
		}
	}

	opts.SetConnectTimeout(5 * time.Second)       // Timeout für die anfängliche Verbindung
	opts.SetPingTimeout(1 * time.Second)          // Timeout für Keep-Alive-Nachrichten
	opts.SetAutoReconnect(true)                   // Automatisch wiederverbinden aktivieren
	opts.SetMaxReconnectInterval(10 * time.Second) // Maximales Intervall zwischen Wiederverbindungsversuchen
	opts.SetWill(topic, "{offline}", false)

	client := mqtt.NewClient(opts)

	// Establish connection to the broker
	fmt.Println("Attempting to connect to the MQTT broker...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Error on first connection to MQTT broker: %v\n", token.Error())
		// If the first connection fails, we will keep trying to connect thanks to SetAutoReconnect(true).
		// systemd will restart the program if it crashes or fails to connect.
	}

	// Publish "online" after successful connection
	if token := client.Publish(statusTopic, 0, false, onlineMessage); token.Wait() && token.Error() != nil {
		fmt.Printf("Error publishing online status to %s: %v\n", onlineMessage, token.Error())
	} else {
		fmt.Printf("Successfully published online status to '%s'\n", statusTopic)
	}

	// Subscribe to topic
	if token := client.Subscribe(topic, 0, messageHandler); token.Wait() && token.Error() != nil {
		fmt.Printf("Error subscribing to topic %s: %v\n", topic, token.Error())
		// If subscription fails, disconnect and exit the program
		client.Disconnect(250)
		os.Exit(1)
	}
	fmt.Printf("Successfully subscribed to topic '%s'\n", topic)

	// Keep the program running and wait for signals
	// We use a channel to wait for OS signals (e.g., SIGINT for Ctrl+C, SIGTERM for systemd)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait until a signal is received
	<-sigChan
	fmt.Println("Shutting down MQTT client...")

	// Verbindung sauber trennen
	client.Disconnect(250) // 250ms Timeout zum Senden ausstehender Nachrichten
	fmt.Println("Client getrennt.")
	os.Exit(0) // Programm sauber beenden
	// Disconnect cleanly
	client.Disconnect(250) // 250ms timeout to send pending messages
	fmt.Println("Client disconnected.")
	os.Exit(0) // Exit program cleanly
}
