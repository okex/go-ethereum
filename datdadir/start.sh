../build/bin/geth --datadir node init genesis.json
../build/bin/geth --networkid 12345 --http --http.api eth,net,web3,miner,txpool --http.addr "0.0.0.0" --http.port "8545" --gcmode "archive"  --allow-insecure-unlock --vmdebug  --datadir node --unlock 0x8620138A2302cEe40c5888C92E29dB911d3C7CB4  --mine --miner.etherbase 0x8620138A2302cEe40c5888C92E29dB911d3C7CB4
