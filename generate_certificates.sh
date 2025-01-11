#!/bin/bash

openssl req -x509 -newkey rsa:4096 -nodes -out cert.pem -keyout key.pem -days 365 \
-subj "/C=RO/ST=Bucuresti/L=Bucuresti/O=CompaniaMea/CN=exemplu.ro"
