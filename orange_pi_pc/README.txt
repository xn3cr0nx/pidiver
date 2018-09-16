Confirmed working on:

https://www.armbian.com/orange-pi-pc/

tested with:

Armbian Stretch 
mainline kernel 4.14.y or 4.17.y

***not compatible without changes with other Orange-Pis! (including Orange Pi PC2!)***

To get SPI working with the PiDiver following changes are necessary:

Add this to /boot/armbianEnv.txt

overlays=spi-spidev spi-add-cs1
param_spidev_spi_bus=0
param_spidev_spi_cs=1
param_spidev_max_freq=10000000

copy the file from ./boot/sun8i-h3-spi-add-cs1.dtbo to /boot/dtb-4.14.65-sunxi/overlay/
and reboot Orange Pi PC.

