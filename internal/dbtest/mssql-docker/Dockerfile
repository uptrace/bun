FROM mcr.microsoft.com/mssql/server:2019-CU29-ubuntu-20.04

RUN openssl req -x509 -nodes -newkey rsa:2048 -subj '/CN=mssql' -addext "subjectAltName = DNS:mssql" -keyout /etc/ssl/private/mssql.key -out /etc/ssl/certs/mssql.pem -days 7 --set_serial 01
RUN chmod 400 /etc/ssl/private/mssql.key
RUN chmod 400 /etc/ssl/certs/mssql.pem
RUN mkdir -p /var/opt/mssql
COPY mssql.conf /var/opt/mssql/mssql.conf


# Create a config directory
RUN mkdir -p /usr/config
WORKDIR /usr/config

# Bundle config source
COPY . /usr/config

ENTRYPOINT ["./entrypoint.sh"]
