../build/bin/geth --datadir node1 init genesis.json
../build/bin/geth --networkid 12345 --http --http.api eth,net,web3,miner,txpool --http.addr "0.0.0.0" --http.port "8645" --authrpc.port "8552" --syncmode "full" --port "30304" --gcmode "archive"  --allow-insecure-unlock --datadir node1
