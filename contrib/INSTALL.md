# TAKL Daemon Installation

## Manual Start

Run the daemon in foreground:
```bash
takl daemon start
```

Run in background using Linux tools:
```bash
# Using nohup
nohup takl daemon start > /tmp/takl-daemon.log 2>&1 &

# Using screen/tmux
screen -S takl-daemon -d -m takl daemon start
```

Check status:
```bash
takl daemon status
```

Stop the daemon:
```bash
takl daemon stop
```

## systemd (Linux)

### Install as user service

1. Copy the service file:
```bash
mkdir -p ~/.config/systemd/user/
cp contrib/systemd/takl-daemon.service ~/.config/systemd/user/
```

2. Enable and start:
```bash
systemctl --user daemon-reload
systemctl --user enable takl-daemon
systemctl --user start takl-daemon
```

3. Check status:
```bash
systemctl --user status takl-daemon
```

### Install system-wide

1. Copy service file:
```bash
sudo cp contrib/systemd/takl-daemon.service /etc/systemd/system/takl-daemon@.service
```

2. Enable for your user:
```bash
sudo systemctl daemon-reload
sudo systemctl enable takl-daemon@$USER
sudo systemctl start takl-daemon@$USER
```

## launchd (macOS)

1. Copy the plist file:
```bash
cp contrib/launchd/com.takl.daemon.plist ~/Library/LaunchAgents/
```

2. Load the service:
```bash
launchctl load ~/Library/LaunchAgents/com.takl.daemon.plist
```

3. Check if running:
```bash
launchctl list | grep takl
```

4. Unload if needed:
```bash
launchctl unload ~/Library/LaunchAgents/com.takl.daemon.plist
```

## Docker

Run the daemon in a container:
```bash
docker run -d \
  --name takl-daemon \
  -v ~/.takl:/root/.takl \
  -v /var/run/docker.sock:/var/run/docker.sock \
  takl/takl daemon start --foreground
```

## Development Mode

For development, run in foreground with verbose output:
```bash
takl daemon start --foreground --verbose
```

This shows all requests and operations in real-time.