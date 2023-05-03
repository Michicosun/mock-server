#!/bin/bash

echo -e "\nSetting up nginx..."
if [[ $(systemctl is-active nginx) == active ]]; then
    echo "Nginx is installed and running"
else
    echo "Nginx is not started. Trying starting"
    if sudo systemctl start nginx; then
        echo "Nginx started"
    else
        echo "Failed to start nginx. Trying to install"

        sudo apt-get update
        sudo apt-get install nginx -y

        sudo systemctl start nginx
        if [ $? -eq 0 ]; then
            echo "Nginx finally installed and running!"
        else
            echo "Unable to deploy nginx, failing..."
            exit 1
        fi
    fi
fi

echo "Copying nginx config file"
if sudo cp nginx.conf /etc/nginx/sites-available/mock_server; then
    echo "Successfully"
else
    echo "Failed to copy nginx config"
    exit 1
fi

echo "Linking config file to sites-enabled"
if sudo ln -sf /etc/nginx/sites-available/mock_server /etc/nginx/sites-enabled/; then
    echo "Sucessfully"
else
    echo "Failed to link nginx config"
    exit 1
fi

echo "Restarting nginx to load new configuration"
sudo systemctl restart nginx
if [ $? -eq 0 ]; then
    echo "New configuration successfully loaded"
else
    echo "Invalid configuration, maybe syntax error"
    sudo nginx -t
    exit 1
fi


STATIC_DIR=/var/www/mock_server/html
if test -d "$STATIC_DIR"; then
    echo "Frontend static directory exists. Seems frontend is deployed"
else
    echo "No frontend statis directory. Frontend not available"
fi


echo -e "\nSetting up docker services"
if sudo docker compose up -d; then
    echo "Docker services started"
else
    echo "Failed to start docker services"
    exit 1
fi


echo -e "\nSetting up systemd mock-server service..."

echo "Building mock-server"
if go build -o mock-server ../cmd/server/main.go; then
    echo "Successfully built"
else
    echo "Failed to build"
    exit 1
fi

echo "Moving binary to /usr/local/bin"
if sudo mv mock-server /usr/local/bin/mock-server; then
    echo "Binary successfully moved"
else
    echo "Failed to move binary"
    exit 1
fi

echo "Copying service config"
if sudo cp mock-server.service /etc/systemd/system/mock-server.service; then
    echo "Successfully copied to /etc/systemd/system/mock-server.service"
else
    echo "Failed to copy mock-server.service"
    exit 1
fi

echo "Reloading systemd daemon"
sudo systemctl daemon-reload
echo "Starting mock-server.service"
sudo systemctl start mock-server.service

echo "Wait 3 seconds until started"
sleep 3

echo "Try ping mock-server"
if curl "$(curl ifconfig.me)/api/ping"; then
    echo -e "\nMock server successfully deployed and running"
else
    echo "Failed to start Mock server :("
    exit 1
fi
