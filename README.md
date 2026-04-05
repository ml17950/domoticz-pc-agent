# domoticz-pc-agent

This repository contains a Go program designed to connect to an MQTT broker, subscribe to a specified topic (`domoticz/atest`), and log received messages. It can be configured to run as a background service on your system.

## Go MQTT Client

### Configuration

The MQTT client's behavior is controlled by a configuration file named `domoticz-pc-agent.ini`. This file should be placed in the same directory as the executable (`/home/martin/Coding/go-domoticz-pc-agent/`).

**File:** `domoticz-pc-agent.ini`

```ini
[mqtt]
broker_address = 192.168.178.250
port = 1883
# username = your_mqtt_username  ; Uncomment and fill if your broker requires authentication
# password = your_mqtt_password  ; Uncomment and fill if your broker requires authentication

[domoticz]
idx = 123 ; Example Domoticz device index (replace with your actual value)
type = Device ; Example Domoticz device type (replace with your actual value)
```

**Settings:**
*   **`[mqtt]` Section:**
    *   `broker_address`: The IP address or hostname of your MQTT broker.
    *   `port`: The port your MQTT broker is listening on (e.g., `1883` for unencrypted, `8883` for TLS).
    *   `username`: (Optional) Your MQTT username. Uncomment and set if required.
    *   `password`: (Optional) Your MQTT password. Uncomment and set if required.
*   **`[domoticz]` Section:**
    *   `idx`: The Domoticz device index.
    *   `type`: The Domoticz device type.

### Installation & Build

1.  **Get the Go code:**
    Ensure you have the Go toolchain installed. Navigate to your project directory (`/home/martin/Coding/go-domoticz-pc-agent/`) in your terminal.
    Save the Go program code into `main.go` as provided in the setup instructions.

2.  **Install Dependencies:**
    Install the necessary MQTT client library:
    ```bash
    go get github.com/eclipse/paho.mqtt.golang
    go get gopkg.in/ini.v1
    ```

3.  **Build the Executable:**
    Compile the Go program:
    ```bash
    go build -o mqtt-client main.go
    ```
    This will create an executable file named `mqtt-client` in the current directory.

### Systemd Service Setup (for Linux)

To run the MQTT client automatically in the background on system startup, you can use `systemd`.

1.  **Create Service File:**
    Create a new file named `mqtt-client.service` in `/etc/systemd/system/` (you'll need `sudo` privileges).
    ```bash
    sudo nano /etc/systemd/system/mqtt-client.service
    ```
    Paste the following content into the file, **replacing `martin` with your actual username if it's different**:

    ```ini
    [Unit]
    Description=Go MQTT Client for Domoticz
    After=network.target

    [Service]
    User=martin
    WorkingDirectory=/home/martin/Coding/go-domoticz-pc-agent
    ExecStart=/home/martin/Coding/go-domoticz-pc-agent/mqtt-client
    Restart=on-failure
    RestartSec=5 # Wait 5 seconds before restarting

    [Install]
    WantedBy=multi-user.target
    ```

2.  **Reload Systemd and Enable/Start Service:**
    After saving the service file, run these commands:
    ```bash
    sudo systemctl daemon-reload
    sudo systemctl enable mqtt-client.service
    sudo systemctl start mqtt-client.service
    ```

3.  **Check Status:**
    You can verify the service status and view logs:
    ```bash
    sudo systemctl status mqtt-client.service
    journalctl -u mqtt-client.service -f # Press Ctrl+C to exit
    ```

## Example Usage (MQTT Commands)

The following `mosquitto_pub` commands demonstrate how to send messages to your MQTT broker that this client might receive or interact with.

```bash
# Example: Sending an 'On' command to a light switch (assuming Domoticz integration)
mosquitto_pub -h 192.168.178.250 -t domoticz/in -m "{"command": "switchlight", "idx": 153, "switchcmd": "On" }"

# Example: Sending an 'offline' status message
mosquitto_pub -h 192.168.178.250 -t "domoticz/in" -m "offline" --will-topic "domoticz/in" --will-payload "offline" --will-retain
```