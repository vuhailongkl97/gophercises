# USAGE
+ update your token and your chat's channelID in /etc/config.yaml  

# APIs
## localhost
* Request to send image with its location  
`curl http://localhost:1234/updated -X POST -d "usecases.drawio.png"`

## In a discord chat channel

* `!enable` disable app inside localhost
* `!disable` enable app inside localhost

# Log
Location: /tmp/serveHTTP

# Install 
``` 
sudo cp iot-discord-proxy /usr/bin
sudo cp iot-discord-proxy.service /etc/systemd/system/iot-discord-proxy.service  
sudo chmod 640 /etc/systemd/system/iot-discord-proxy.service
sudo systemctl daemon-reload
systemctl enable iot-discord-proxy.service
systemctl start iot-discord-proxy.service
```

