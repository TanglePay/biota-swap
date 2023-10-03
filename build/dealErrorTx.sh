rm -rf ./biota-swap
git clone https://github.com/TanglePay/biota-swap
cd biota-swap
git checkout error_deal
go build -ldflags "-w -s"
cp bwrap ../bdeal_err
cd ..

if [ ! -f "./config/smpc_k" ];then
    echo -e "\e[31m !!! panic : Must cp the smpc_k file to the path of ./config/"
    exit
fi

./bdeal_err ETH sETH 0x090a8850aec72e4ce6a26a0d3144eb94e0587c11dc31a0f9b3bfdd158c7121ac
./bdeal_err WBTC sBTC 0x2c65f1f6b0ac16c235238e72e5fe000ff62a1f215bd702a8bcd08fb95000edf7