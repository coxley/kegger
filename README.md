# Kegger

Drinking from a keg is fun.

Whether it's beer, coffee, or prosecco you're having a good time. Do you know
what the opposite of a good time is? Going to pour your favorite potion, and
being left with a foamy, 1/4 glass because you didn't realize it was out. On
top of that, your supplier demands a 2-3 day lead time to renew your fix.

Let's get ahead of the problem and maybe gamify it along the way.

# Photos and Videos

TBD...

# Isn't this pretty... niche?

You betcha.

Kegger's roots come from Facebook Hackathons. Not only did most offices have a
keg setup of some kind — many teams in Menlo Park had a bar identity of their
own for happy hours. Some had an FB group for "What's on Tap", some a chat
thread, and even one would laser-etch your username on a pint glass for your
first visit.

One team had a web portal, but in [XKCD 927 fashion](https://xkcd.com/927/)[1]
we decided to be **even more ambitious**.[2]

One of our colleagues started by hacking a Wii Fit to upload the weight of our
mini-fridge in a graph. While this allowed us to get emails when it was running
low, anyone can look inside a fridge. The real "problems" were the kegs.

So with a bit of planning for the next Hackathon, we sourced some FDA-approved
flow meters, integrated our badge readers into it, and got to work. Our web
development skills then were attrocious — but the data was now there. You could
_see_ that the keg was ~16oz emptier with each pour. You could be greeted with
your name when badging in, seeing your photo.

Kegger aims to take something silly like this and put it in the hands of other
geeky friend groups out there.

[1] <img src="https://imgs.xkcd.com/comics/standards_2x.png" width="500"/>
[2] ...by getting it running in two offices and stopping there once COVID hit

# Running the software

```
$ make all
$ sudo make install
$ kegger -tap 1:GPIO2 -tap 2:GPIO3 -frontend ./www/build/
```

# Materials

You can get this to work in a tons of set-ups — this is just the one
that I've done. The basics are a flow meter, a hose-to-NPT adapter, an NPT coupler
to connect them (reducer if the sizes don't match), spare wires and solder,
and a Raspberry Pi. Throw in an old tablet to run the web-page 24/7 and you're
good-to-go.

You can definitely use an ESP or Arduino — you'll just have to write the code
for it.

- Flow Controller
    - [Gems FT-330](https://www.gemssensors.com/search-products/product-details/ft-330-series-turbine-flow-sensor-226000) (Overkill and expensive... but works well)
    - [Cheaper option](https://www.amazon.com/DIGITEN-0-3-6L-Flowmeter-Counter-Connect/dp/B072JVL5VG?ref_=ast_sto_dp)
    - [And another](https://www.amazon.com/DIGITEN-Sensor-Effect-Flowmeter-Counter/dp/B07QNN2GRV/ref=sr_1_10?c=ts&keywords=Flow%2BSensors&qid=1657573740&s=industrial&sr=1-10&ts_id=306928011&th=1)
    - Whichever you go with, you'll need to adjust the `-ppg` value based on their specsheet.
    - You need one per keg.

- Hose adapter
    - [3/16 hose to 1/2 MIP (NPT)](https://www.homedepot.com/p/LTWFITTING-3-16-in-ID-Hose-Barb-x-1-2-in-MIP-Lead-Free-Brass-Adapter-Fitting-5-Pack-HFLF39183805/313323908)
    - Make sure the inner-diameter (ID) matches your keg's output line.
    - You need two per keg.

- Coupler
    - [Reducer](https://www.homedepot.com/p/Southland-1-2-in-x-3-8-in-Black-Malleable-Iron-FPT-x-FPT-Reducing-Coupling-FItting-521-332HN/100135006)
    - Make sure the sizes align with your flow controller and hose adapter.
    - You need two per keg.

- Raspberry Pi
    - [Zero W](https://www.adafruit.com/product/5291)
    - [RPi 4 Model B](https://www.raspberrypi.com/products/raspberry-pi-4-model-b/)
    - Going with a wireless-capable option makes it so much more convenient.
    - You need one per bar / area.

- Spare wires and solder
    - Really anything works. I like to pull networking (CAT5) cable apart and
      use strands from it. Then you can solder jumper connectors on the end for
      easy maintenance.

    - If you have several kegs, you can share one of each 5v and ground lines for each flow meter. This reduces the number of pins you need to use on the RPi.
