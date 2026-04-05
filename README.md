# domoticz-pc-agent

This repository contains a Go program designed to connect to an MQTT broker, subscribe to a specified topic (`domoticz/atest`), and log received messages. It can be configured to run as a background service on your system.

## Go MQTT Client

### Configuration

The MQTT client's behavior is controlled by a configuration file named `domoticz-pc-agent.ini`. This file should be placed in the same directory as the executable.

**File:** `domoticz-pc-agent.ini`

```ini
[mqtt]
; Broker address/port
broker_address = 192.168.x.y
port = 1883
; Username/Password (Uncomment and fill if your broker requires authentication)
;username = your_mqtt_username
;password = your_mqtt_password

[domoticz]
; Domoticz device id (replace with your actual value)
idx = 111
; Domoticz type (Switch or Sensor)
type = Switch
```

### Installation & Build

**Get the Go code:**  
Ensure you have the Go toolchain installed.  
TODO git command

**Install Dependencies:**  
Install the necessary MQTT client library:  
```bash
go get github.com/eclipse/paho.mqtt.golang
go get gopkg.in/ini.v1
```

**Build the Executable:**  
Compile the Go program:
```bash
./build.sh
```
 This will create an executable file named `domoticz-pc-agent` in the current ./bin/ directory.

### Systemd Service Setup (for Linux)

To run the MQTT client automatically in the background on system startup, you can use `systemd`.

**Copy binary and create config:**  
Copy/create needed files in `/usr/local/bin/` (you'll need `sudo` privileges).
```bash
sudo cp ./bin/domoticz-pc-agent /usr/local/bin/
sudo nano /usr/local/bin/domoticz-pc-agent.ini
```

**Create Service File:**  
Create a new file named `domoticz-pc-agent.service` in `/etc/systemd/system/` (you'll need `sudo` privileges).
```bash
sudo nano /etc/systemd/system/domoticz-pc-agent.service
```
Paste the following content into the file, **replacing `martin` with your actual username if it's different**:

```ini
[Unit]
Description=Go MQTT Client for Domoticz
After=network.target

[Service]
User=martin
WorkingDirectory=/usr/local/bin
ExecStartPre=/bin/sleep 5
ExecStart=/usr/local/bin/domoticz-pc-agent
Restart=on-failure
RestartSec=5 # Wait 5 seconds before restarting

[Install]
WantedBy=multi-user.target
```

**Reload Systemd and Enable/Start Service:**  
The `After=network.target` directive ensures the service starts only after a network connection is established.
```bash
sudo systemctl daemon-reload
sudo systemctl enable domoticz-pc-agent.service
sudo systemctl start domoticz-pc-agent.service
```

**Check Status:**  
You can verify the service status and view logs:
```bash
sudo systemctl status domoticz-pc-agent.service
journalctl -u domoticz-pc-agent.service -f # Press Ctrl+C to exit
```

## Example Usage (MQTT Commands)

The following `mosquitto_pub` commands demonstrate how to send messages to your MQTT broker that this client might receive or interact with.

```bash
# Example: Sending an 'On' command to a light switch (assuming Domoticz integration)
mosquitto_pub -h 192.168.178.250 -t domoticz/in -m "{"command": "switchlight", "idx": 153, "switchcmd": "On" }"

# Example: Sending an 'offline' status message
mosquitto_pub -h 192.168.178.250 -t "domoticz/in" -m "offline" --will-topic "domoticz/in" --will-payload "offline" --will-retain
```