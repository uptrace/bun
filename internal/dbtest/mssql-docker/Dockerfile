FROM mcr.microsoft.com/mssql/server:2019-CU29-ubuntu-20.04

RUN mkdir -p /var/opt/mssql
RUN openssl req -x509 -nodes -newkey rsa:2048 \
        -subj '/CN=mssql' -addext "subjectAltName = DNS:mssql"  \
        -keyout /var/opt/mssql/mssql.key -out /var/opt/mssql/mssql.pem \
        -days 7 -set_serial 01
RUN chmod 400 /var/opt/mssql/mssql.key && \
    chmod 400 /var/opt/mssql/mssql.pem
COPY mssql.conf /var/opt/mssql/mssql.conf

# Create a config directory
RUN mkdir -p /usr/config
WORKDIR /usr/config

# Bundle config source
COPY . /usr/config

ENTRYPOINT ["./entrypoint.sh"]
