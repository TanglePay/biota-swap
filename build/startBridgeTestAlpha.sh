SmrRpcUrl="https://json-rpc.evm.shimmer.network"
SmrWssUrl="wss://ws.json-rpc.evm.shimmer.network"
IotaRpcUrl="https://api.stardust-mainnet.iotaledger.net"
IotaWssUrl="https://api.stardust-mainnet.iotaledger.net"
SubGroup="3b4b356128ae04509e635fdd3d6d16e86715313969f1725e338aed197b140183ef3fd111862b87cdc952107813db0a162a0b8d9ddd36dbe9870ade495ffd13a7"
EthRpcUrl="https://sepolia.infura.io/v3/d76f3ff2954844359b16db013a099e45"
EthWssUrl="wss://sepolia.infura.io/ws/v3/d76f3ff2954844359b16db013a099e45"

cat> config/conf.toml << :EOF
# Version
Version = "1.0.0"
# PendingTime is time of seconds for a tx keep pending status
PendingTime = 300

# NodeUrl is the node url of smpc
# Gid is the subgroup id
# ThresHold is the group rule. It can be "2/3", "3/5", "4/6" ...
# KeyStore is the key of the node of smpc
[Smpc]
NodeUrl = "http://127.0.0.1:5871"
Gid = "$SubGroup"
ThresHold   ="2/3"
KeyStore = "./config/smpc_k"

# The Server config
# DetectCount is the detect count when it request a sign to accept. The DetectTime is the time as seconds between two detect loops.
# AcceptTime is the check time as seconds with one loop.
# AcceptOverTime is the time as seconds. If smpc sign over this time, it should be not accepted.
[Server]
DetectCount = 60
DetectTime = 10
AcceptTime = 30
AcceptOverTime = 600

# database driver is mysql
# the dabasebase name is "smpc" and the table to see the "readme"
[Db]
Host = "127.0.0.1"
Port = "3306"
DbName = "smpc"
Usr="test1"
Pwd="123456"

[TxErrorRecord]
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
Contract = "0x3Ede34eF08d7822b28D31A8aa864F9ac7FD3566A"
ScanEventType = 0
TimePeriod = 3600

# Tokens contain "IOTA", sIOTA", "ETH", "sETH", "WBTC", "sBTC"
# Symbol is the unique
# ScanEventType, 0: listen event as websockt or mqtt; 1: scan block to get event logs.
# MultiSignType, 0 is contract multiSign, 2 is smpc multiSign
[[Tokens]]
Symbol = "IOTA"
NodeRpc = "$IotaRpcUrl"
NodeWss = "$IotaWssUrl"
ScanEventType = 0
MultiSignType = 2
# iota1qryydwght5fkguktsy9rfzarqt9gx3rvpzzkzfnpq2aalqn6mvnpq0d8wjm, atoi1qryydwght5fkguktsy9rfzarqt9gx3rvpzzkzfnpq2aalqn6mvnpqgrk0gk
PublicKey = "8786dc216e64b7f20c8ccb45ca0474d4e9819734cfe60e25c7aacae1bc8bcd6f"
MinAmount = 1000000

[[Tokens]]
Symbol = "ETH"
NodeRpc = "$EthRpcUrl"
NodeWss = "$EthWssUrl"
ScanEventType = 0 
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0x54773f9c01B9A7E70e7B62EFf871BC5E310F1910"
KeyStore = "./config/k"
MinAmount = 1

[[Tokens]]
Symbol = "WBTC"
NodeRpc = "$EthRpcUrl"
NodeWss = "$EthWssUrl"
ScanEventType = 0
ScanMaxHeight = 1000
MultiSignType = 0
Contract = "0x1b562bf60d69E17e7F6C7BEec16FB8FFB419EB20"
KeyStore = "./config/k"
MinAmount = 1

# target tokens
[[Tokens]]
Symbol = "sIOTA"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
MultiSignType = 0
Contract = "0x99Bd15Ca1F52633b2652C3F13F6D7026ce88b7bF"
KeyStore = "./config/k"
MinAmount = 1000000

[[Tokens]]
Symbol = "sETH"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
MultiSignType = 0
Contract = "0x54773f9c01B9A7E70e7B62EFf871BC5E310F1910"
KeyStore = "./config/k"
MinAmount = 1

[[Tokens]]
Symbol = "sBTC"
NodeRpc = "$SmrRpcUrl"
NodeWss = "$SmrWssUrl"
ScanEventType = 0
MultiSignType = 0
Contract = "0x1b562bf60d69E17e7F6C7BEec16FB8FFB419EB20"
KeyStore = "./config/k"
MinAmount = 1

# Pairs is the bridge pair. 
# SrcToken to DestToken. They must beed in the "Tokens".
[[Pairs]]
SrcToken = "IOTA"
DestToken = "sIOTA"

[[Pairs]]
SrcToken = "ETH"
DestToken = "sETH"

[[Pairs]]
SrcToken = "WBTC"
DestToken = "sBTC"

:EOF

./bwrap_main -d