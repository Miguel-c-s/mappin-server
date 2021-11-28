#!/bin/bash

docker run -it -d --name mappin-server --hostname mappin-server --net internal --restart unless-stopped mappin-server:1.0
