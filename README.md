# Golang-PiDiver-Lib for the PiDiver

IOTA PoW needs a lot of computational power which makes sending transactions on smaller microcontrollers (like ARM) very slow. One of the main reasons is that the innerst loop of Curl-P81 can’t be computed very efficient on general purpose CPUs. Even modern CPUs with SIMD extension (like SSE or AVX) are heavily restricted when it comes to true parallel calculations.

This project is about development of a hardware accelerator for doing PoW on embedded Systems and low-end PCs efficiently.

The PiDiver archives about 15.8MH/s and it is able to do PoW faster (about x5.8) than the SSE-optimized dcurl library on a quad-core i5 PC but only needs 2W instead of 100W.

Statistically, 25% of all nonces are found within 87ms, 50% within 200ms and 75% within 422ms. That gives an average of 3.33 TX/s or 0.66 TPS (bundle with 5TX).

Comparing to an Raspberry Pi, that's a speed-up of >200.

Here you can find two project sites for the PiDiver project:

https://ecosystem.iota.org/projects/iota-pearl-diver-pow-fpga-raspberry-pi

http://microengineer.eu/2018/04/25/iota-pearl-diver-fpga/


If you think, the project is worth supporting, please consider to leave me a donation at:

LLEYMHRKXWSPMGCMZFPKKTHSEMYJTNAZXSAYZGQUEXLXEEWPXUNWBFDWESOJVLHQHXOPQEYXGIRBYTLRWHMJAOSHUY

Discord: pmaxuw#8292

Thank you very much :)


# License

This project is licensed under the MIT-License (https://opensource.org/licenses/MIT)
