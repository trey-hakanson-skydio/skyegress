# SkyEgress

## Overview

- sky egress server
  - components
    - http server
    - rtsp server
  - flow
    - start egress request made to http server (we could use the rtsp server's announce, but that seems overly complicated)
    - connect to server
    - add track


dev flow:

```sh
# join dev cluster
source .env
livekit-cli join-room \
  --publish-demo \
  --room devroom \
  --identity publisher \
  --url "wss://$LIVEKIT_URL"

# start server
source .env
go run main.go serve

# start an egress session
go run main.go client start \
  --room-name devroom \
  --track-name demo

# stop egress session
go run main.go client stop \
  --room-name devroom \
  --track-name demo
```

## Questions

- Does it make more sense to create an egress component that joins the room and feeds packets into a gortsplib client, and then still run rtsp simple server?
