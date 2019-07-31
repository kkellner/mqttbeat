# Mqttbeat

Welcome to Mqttbeat for elastic stack version 8.0.0 (works for 7.2.0)

## Getting Started with Mqttbeat

### Build

To build the Docker Image for Mqttbeat run the command below.

```
docker build -t tmechen/mqttbeat:8.0.0 .
```

### Run

MQTTBeat is a Docker Image which depends on a mqttbeat.yml and field.yml config file to be mounted at /config/
Therefore an example docker-compose file looks like this:

```
version: "3.7"
services:
  mqttbeat:
    image: tmechen/mqttbeat:8.0.0
    container_name: elk_mqttbeat
    restart: always
    depends_on:
      - elasticsearch
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - /elk/mqttbeat/config/:/config/:ro
```

### Config

* Example mqttbeat.yml looks like this:
```
mqttbeat:
  broker_url: "tcp://broker:1883"
  topics_subscribe:
    - example/#?2

output.elasticsearch:
  hosts: ["elasticsearch"]
  ```
  
* Example field.yml looks like this:
```
- key: message
  title: "Mqtt message"
  description: >
    A message from a mqtt client
  fields:
    - name: topic
      required: false
      description: >
        The topic corresponding to the message, basically an url
```