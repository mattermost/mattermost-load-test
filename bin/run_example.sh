mcreate users -r -n 5 | mcreate teams -r -n 2 | mcreate channels -n 10 | mmanage login | mmanage jointeam | mmanage joinchannel > state.json
cat state.json | loadtest active
