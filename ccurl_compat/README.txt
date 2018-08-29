libccurl-compatible library for use with the USBDiver

compile with:

go build -o libccurl.so -buildmode=c-shared libccurl.go 

then copy the .so file to the location of the original libccurl.so file - e.g.

sudo cp libccurl.so "/opt/IOTA Wallet/resources/ccurl/lin64/libccurl.so"

you also need the fpga core file. Also copy it to the location:

sudo cp ../pidiver1.1.rbf "/opt/IOTA Wallet/resources/ccurl/lin64/pidiver1.1.rbf"

