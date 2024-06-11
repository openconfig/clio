### Populating Certificates

The Dockerfile will copy the contents of this directory to ```/certs/```.
To, e.g., generate a self-signed TLS certificate & key pair for testing, run:

    # Create private key for server.
    openssl genrsa -out cert.key 2048 

    # Create the certificate signing request.
    openssl req -new -key cert.key -out cert.csr 

    # Sign the certificate using the private key and CSR, valid for 100 years.
    openssl x509 -req -days 36500 -in cert.csr -signkey cert.key -out cert.crt

    # Cleanup the no-longer-needed signing request.
    rm cert.csr

Afterward, you can supply the cert and key file to clio in the config like this:

    exporters:
      gnmi:
        cert_file: /certs/cert.crt
        key_file: /certs/cert.key