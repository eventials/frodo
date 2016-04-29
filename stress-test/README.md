# load testing

This code ains to test many users connected to Frodo.

## Install

Ensure to use Python 2.7.

`pip install -r requirements.txt`

## Run

Over command line run:

`locust -f locustfile.py -H http://<FRODO-IP-DOMAIN> -c 100000 -r 1000 -n 10000 --no-web`

If you want to use the web version run:

`locust -f locustfile.py -H http://<FRODO-IP-DOMAIN>`
