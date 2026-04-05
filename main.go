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

// connectHandler wird aufgerufen, wenn die Verbindung zum MQTT-Broker erfolgreich ist.
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Erfolgreich mit dem MQTT-Broker verbunden!")
}

// connectLostHandler wird aufgerufen, wenn die Verbindung zum MQTT-Broker unerwartet verloren geht.
var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Verbindung verloren: %v\n", err)
	// systemd wird das Programm neu starten, falls es abstürzt.
}

// messageHandler wird aufgerufen, wenn eine Nachricht auf einem abonnierten Thema empfangen wird.
var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Nachricht empfangen auf Thema: %s\n", msg.Topic())
	fmt.Printf("Payload: %s\n", msg.Payload())
	// Hier könnten Sie weitere Logik hinzufügen, z.B. die Nachricht an Domoticz weiterleiten
	// basierend auf den aus der INI-Datei gelesenen Idx und Type.
}

func main() {
	// Konfigurationsdatei laden
	cfgFile := "domoticz-pc-agent.ini"

	// 1. Lade die INI-Datei in ein ini.File Objekt
	file, err := ini.Load(cfgFile)
	if err != nil {
		fmt.Printf("Fehler beim Laden der Konfigurationsdatei '%s': %v\n", cfgFile, err)
		os.Exit(1)
	}

	// 2. Mappe die geladenen Konfiguration auf die Config-Struktur
	var cfg Config
	err = file.MapTo(&cfg)
	if err != nil {
		fmt.Printf("Fehler beim Mappen der Konfiguration auf die Struktur: %v\n", err)
		os.Exit(1)
	}

	// Überprüfen, ob essenzielle Konfiguration vorhanden ist
	if cfg.MQTT.BrokerAddress == "" || cfg.MQTT.Port == "" {
		fmt.Println("Fehler: MQTT Broker-Adresse und Port müssen in der INI-Datei gesetzt sein.")
		os.Exit(1)
	}
	// Ausgabe der gelesenen Konfiguration (optional, zur Fehlersuche)
	fmt.Printf("MQTT Broker: %s:%s\n", cfg.MQTT.BrokerAddress, cfg.MQTT.Port)
	if cfg.MQTT.Username != "" {
		fmt.Printf("MQTT Benutzer: %s\n", cfg.MQTT.Username)
	}
	if cfg.Domoticz.Idx != "" {
		fmt.Printf("Domoticz Index: %s\n", cfg.Domoticz.Idx)
	}
	if cfg.Domoticz.Type != "" {
		fmt.Printf("Domoticz Typ: %s\n", cfg.Domoticz.Type)
	}

	// MQTT Client Konfiguration aus der INI-Datei erstellen
	brokerURL := fmt.Sprintf("tcp://%s:%s", cfg.MQTT.BrokerAddress, cfg.MQTT.Port)
	clientID := "domoticz-pc-agent"
	topic := "domoticz/atest"             // Beibehalten oder aus INI lesen, falls gewünscht

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetDefaultPublishHandler(messageHandler) // Nachrichten für Abonnements verarbeiten
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	// Nur Username und Passwort setzen, wenn sie in der INI-Datei vorhanden sind
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

	// Verbindung zum Broker herstellen
	fmt.Println("Versuche, eine Verbindung zum MQTT-Broker herzustellen...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Printf("Fehler bei der ersten Verbindung zum MQTT-Broker: %v\n", token.Error())
		// Wenn die erste Verbindung fehlschlägt, versuchen wir weiter zu verbinden dank SetAutoReconnect(true).
		// systemd wird das Programm neu starten, falls es abstürzt oder die Verbindung nicht zustande kommt.
	}

	// Veröffentliche "{online}" nach erfolgreicher Verbindung
	onlineTopic := fmt.Sprintf("%s/status", topic) // Use the existing topic as a base
	if token := client.Publish(onlineTopic, 0, false, "{online}"); token.Wait() && token.Error() != nil {
		fmt.Printf("Fehler beim Veröffentlichen des Online-Status auf %s: %v\n", onlineTopic, token.Error())
	} else {
		fmt.Printf("Erfolgreich Online-Status auf '%s' veröffentlicht.\n", onlineTopic)
	}

	// Thema abonnieren
	if token := client.Subscribe(topic, 0, messageHandler); token.Wait() && token.Error() != nil {
		fmt.Printf("Fehler beim Abonnieren des Themas %s: %v\n", topic, token.Error())
		// Wenn das Abonnieren fehlschlägt, trennen wir die Verbindung und beben das Programm
		client.Disconnect(250)
		os.Exit(1)
	}
	fmt.Printf("Erfolgreich zum Thema '%s' abonniert.\n", topic)

	// Das Programm am Laufen halten und auf Signale warten
	// Wir verwenden einen Channel, um auf Betriebssystem-Signale zu warten (z.B. SIGINT für Strg+C, SIGTERM für systemd)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Warten, bis ein Signal empfangen wird
	<-sigChan
	fmt.Println("Herunterfahren des MQTT-Clients...")

	// Verbindung sauber trennen
	client.Disconnect(250) // 250ms Timeout zum Senden ausstehender Nachrichten
	fmt.Println("Client getrennt.")
	os.Exit(0) // Programm sauber beenden
}
