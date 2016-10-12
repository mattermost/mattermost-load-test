#!/bin/bash

read -p "Did you configure loadtestconfig.json with your connection and admin user info?"
read -p "Is this a blank DB other than your admin user?"
read -p "Have you changed the server to be and open server in config.json?"
read -p "Have you changed the max users per team to be at least 30000?"

mcreate users | mcreate teams | mcreate channels | mmanage login | mmanage jointeam | mmanage joinchannel > state.json

echo "You may now run ./run.sh"
