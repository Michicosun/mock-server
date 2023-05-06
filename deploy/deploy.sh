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

echo "Creating working directory if not exists"
if test -d /etc/mock-server; then
    echo "Working directory already exists!"
else
    echo "Creating new one"
    sudo mkdir /etc/mock-server

    if [ $? -neq 0 ]; then
        echo "Failed to create working directory"
        exit 1
    fi
fi

sudo mkdir -p /etc/mock-server/configs

echo "Copying golang binary config to working directory"
if sudo cp ../configs/config.yaml /etc/mock-server/configs/; then
    echo "Successfully copied"
else
    echo "Failed to copy golang binary config to /etc/mock-server/configs/"
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
sudo systemctl restart mock-server.service

echo "Wait 10 seconds until started"
sleep 10

echo "Try ping mock-server locally"
if curl -s http://localhost:1337/api/ping; then
    echo -e "\nMock server running locally"
else
    echo "Failed to start Mock server :("
    exit 1
fi

echo "Try ping mock-server through nginx"
if [[ $(curl -s "$(curl -s ifconfig.me)/api/ping" | head -n 1 | cut -d' ' -f2) != 200 ]]; then
    echo -e "\nMock server successfully deployed and running. Available on http://$(curl -s ifconfig.me)"
else
    echo "Failed to start Mock server :("
    exit 1
fi
