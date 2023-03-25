https://appliedgo.net/networking/
https://github.com/billglover/tcpip
https://betterprogramming.pub/build-a-tcp-connection-pool-from-scratch-with-go-d7747023fe14
https://www.saminiir.com/lets-code-tcp-ip-stack-1-ethernet-arp/

https://github.com/pandax381/microps
https://github.com/pandax381/lectcp
https://github.com/hsheth2/gonet/blob/master/tap_setup.sh

https://terassyi.net/posts/2020/04/01/arp.html

goal:

リクエストを自作した TCP/IP プロトコルを介して処理できること

task:

アプリケーション層, プレゼンテーション層, セッション層 HTTP...
トランスポート層 TCP...
ネットワーク層 Ipv4, ARP...
データリンク層 イーサネット...

## データリンク

MAC アドレス: データリンクに接続しているノードを識別する

### イーサネット と ARP

イーサネット

> LAN や WAN を構成する有線ローカルエリアネットワークの主流な通信規格
> https://ja.wikipedia.org/wiki/%E3%82%A4%E3%83%BC%E3%82%B5%E3%83%8D%E3%83%83%E3%83%88

ローカルホスト間での通信のためあまり気にしなくて良さそう
