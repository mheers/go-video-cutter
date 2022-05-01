# Go-video-cutter

> Cuts videos using Go FFmpeg Bindings from github.com/3d0c/gmf

## Cut from growing file

- Start a stream forwarder

```bash
srt-live-transmit srt://:1234 srt://:4201 -v
```

- Start a stream:

```bash
ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 -pix_fmt yuv420p -vf "drawtext=fontfile=/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf: 'text=%{localtime\:%T}': fontcolor=white@0.8: x=7: y=70: fontsize=(h/20)" -c:v libx264 -preset ultrafast  -f mpegts "srt://127.0.0.1:1234?pkt_size=1316"
```

- Start a listener that records to growing file:

```bash
ffmpeg -probesize 32 -analyzeduration 0 -i "srt://127.0.0.1:4201?transtype=live&latency=0&recv_buffer_size=0" -c copy -f mpegts - | tee -a /tmp/out.ts
```

### TODO

- [ ] add utc timestamps to the growing file
