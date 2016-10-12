#!/bin/bash

read -p "Did you run setup.sh?"

cat state.json | loadtest listenandpost
