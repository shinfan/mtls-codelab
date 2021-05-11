# PKCS11 Codelab using Google Golang Clients

This is a codelab that demonstrantes the sample use case of using Google clients (golang) with mTLS authentication where the certificate is loaded with PKCS11 interface.

### Pre-requisite

- This codelab is supposed to be ran at Debian GNU/Linux.
- Your environment should have golang installed (minimum version 1.16)

## Install Google API Clients

### Download the codelab and its dependencies

```bash
# Clone the codelab repo
git clone git@github.com:shinfan/mtls-codelab.git

# Clone the Google API clients repo
git clone git@github.com:shinfan/google-api-go-client.git

cd mtls-codelab.git
```

Modify the go.mod to replace the dependency to the your local google client repo which has the PKCS11 support:

```
replace google.golang.org/api => /usr/local/google/home/shinfan/go/src/google-api-go-client
```

## Install PKCS11 support and Verify with OpenSSL

### Install openssl with pkcs11 

First install openssl with its [PKCS11 engine](https://github.com/OpenSC/libp11#openssl-engines).

```bash
# add to /etc/apt/sources.list
  deb http://http.us.debian.org/debian/ testing non-free contrib main

# then
$ export DEBIAN_FRONTEND=noninteractive 
$ apt-get update && apt-get install libtpm2-pkcs11-1 tpm2-tools libengine-pkcs11-openssl opensc -y
```

Note, the installation above adds in the libraries for all modules in this repo (TPM, OpenSC, etc)..you may only need `libengine-pkcs11-openssl` here to verify

Once installed, you can check that it can be loaded:

Set the pkcs11 provider and module directly into openssl (make sure `libpkcs11.so` engine reference exists first!)

- `/etc/ssl/openssl.cnf`
```bash
openssl_conf = openssl_def
[openssl_def]
engines = engine_section

[engine_section]
pkcs11 = pkcs11_section

[pkcs11_section]
engine_id = pkcs11
dynamic_path = /usr/lib/x86_64-linux-gnu/engines-1.1/libpkcs11.so
```

```bash
$ ls /usr/lib/x86_64-linux-gnu/engines-1.1/
afalg.so  libpkcs11.so  padlock.so  pkcs11.la  pkcs11.so

$ openssl engine
  (rdrand) Intel RDRAND engine
  (dynamic) Dynamic engine loading support

$ openssl engine -t -c pkcs11
  (pkcs11) pkcs11 engine
  [RSA, rsaEncryption, id-ecPublicKey]
      [ available ]

      dynamic_path = /usr/lib/x86_64-linux-gnu/engines-1.1/libpkcs11.so
```

---

### SOFTHSM

SoftHSM is as the name suggests, a sofware "HSM" module used for testing.   It is ofcourse not hardware backed but the module does allow for a PKCS11 interface which we will also use for testing.

First make sure the softhsm library is installed

- [SoftHSM Install](https://www.opendnssec.org/softhsm/)

Setup a config file where the `directories.tokendir` points to a existing folder where softHSM will save all its data (in this case its `./tokens/`)

>> This repo already contains a sample configuration/certs to use with the softhsm token directory...just delete the folder and start from scratch if you want..

```bash
mkdir tokens
```

Edit `softhsm.conf`
and edit the value for `directories.tokendir`

```bash
log.level = DEBUG
objectstore.backend = file
directories.tokendir = /absolute/path/to/pkcs11_signer/misc/tokens/
slots.removable = true
```

Now, make sure that the installation created the softhsm module for openssl:  `[PATH]/libsofthsm2.so`

```bash
openssl engine dynamic \
 -pre SO_PATH:/usr/lib/x86_64-linux-gnu/engines-1.1/libpkcs11.so \
 -pre ID:pkcs11 -pre LIST_ADD:1 \
 -pre LOAD \
 -pre MODULE_PATH:[PATH]/libsofthsm2.so \
 -t -c

  (dynamic) Dynamic engine loading support
  [Success]: SO_PATH:/usr/lib/x86_64-linux-gnu/engines-1.1/libpkcs11.so
  [Success]: ID:pkcs11
  [Success]: LIST_ADD:1
  [Success]: LOAD
  [Success]: MODULE_PATH:[PATH]/libsofthsm2.so
  Loaded: (pkcs11) pkcs11 engine
  [RSA, rsaEncryption, id-ecPublicKey]
      [ available ] 
```

Use [pkcs11-too](https://manpages.debian.org/testing/opensc/pkcs11-tool.1.en.html) which comes with the installation of opensc

```bash
export SOFTHSM2_CONF=/absolute/path/to/pkcs11_signer/misc/softhsm.conf

## init softhsm
pkcs11-tool --module [PATH]/libsofthsm2.so --slot-index=0 --init-token --label="token1" --so-pin="123456"

## Change pin and list token slots
pkcs11-tool --module [PATH]/libsofthsm2.so  --label="token1" --init-pin --so-pin "123456" --pin mynewpin

$ pkcs11-tool --module [PATH]/libsofthsm2.so --list-token-slots
        Available slots:
        Slot 0 (0x51b9d639): SoftHSM slot ID 0x51b9d639
        token label        : token1
        token manufacturer : SoftHSM project
        token model        : SoftHSM v2
        token flags        : login required, rng, token initialized, PIN initialized, other flags=0x20
        hardware version   : 2.6
        firmware version   : 2.6
        serial num         : 11819f2dd1b9d639
        pin min/max        : 4/255

# Create object
pkcs11-tool --module [PATH]/libsofthsm2.so -l -k --key-type rsa:2048 --id 4142 --label keylabel1 --pin mynewpin

$ pkcs11-tool --module [PATH]/libsofthsm2.so  --list-objects
        Using slot 0 with a present token (0x51b9d639)
        Public Key Object; RSA 2048 bits
        label:      keylabel1
        ID:         4142
        Usage:      encrypt, verify, wrap
        Access:     local

### use key to generate random bytes

pkcs11-tool --module [PATH]/libsofthsm2.so  --label="keylabel1" --pin mynewpin --generate-random 50 | xxd -p

### Use openssl module to sign and print the public key (not, your serial number will be different)
export PKCS11_PRIVATE_KEY="pkcs11:model=SoftHSM%20v2;manufacturer=SoftHSM%20project;serial=11819f2dd1b9d639;token=token1;type=private;object=keylabel1?pin-value=mynewpin"
export PKCS11_PUBLIC_KEY="pkcs11:model=SoftHSM%20v2;manufacturer=SoftHSM%20project;serial=11819f2dd1b9d639;token=token1;type=public;object=keylabel1?pin-value=mynewpin"

### Sign and verify
echo "sig data" > "data.txt"
openssl rsa -engine pkcs11  -inform engine -in "$PKCS11_PUBLIC_KEY" -pubout -out pub.pem
openssl pkeyutl -engine pkcs11 -keyform engine -inkey $PKCS11_PRIVATE_KEY -sign -in data.txt -out data.sig
openssl pkeyutl -pubin -inkey pub.pem -verify -in data.txt -sigfile data.sig

### Display the public key
openssl rsa -engine pkcs11  -inform engine -in "$PKCS11_PUBLIC_KEY" -pubout
```

### Generate mTLS certs

At this point, we can use the embedded private keys to generate x509 certificate CSRs. Note: devices can already generate x509 certs associated with  its private key but in this case, we will use an externally generated CA using the private key alone

```bash
git clone https://github.com/salrashid123/ca_scratchpad
cd ca_scratchpad
```

follow the three steps as described in `ca_scratchpad/README.md`:  

- Create Root CA
- Gen CRL
- Create Subordinate CA for TLS Signing

stop after setting up the CA

Now that the CA is setup, we need to create a CSR using the private key in the SoftHSM

```bash
export SOFTHSM2_CONF=/absolute/path/to/pkcs11_signer/misc/softhsm.conf
cd csr/

## you will see the CSR based on the private key in SoftHSM (your cert will be different!)
$ go run csr/csr.go 
-----BEGIN CERTIFICATE REQUEST-----
MIIC7DCCAdQCAQAwejELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWEx
FjAUBgNVBAcTDU1vdW50YWluIFZpZXcxDzANBgNVBAoTBkdvb2dsZTETMBEGA1UE
CxMKRW50ZXJwcmlzZTEYMBYGA1UEAxMPcGtjcy5kb21haW4uY29tMIIBIjANBgkq
hkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuGAmiw+2vke+9uy/zstyK56WcASTxL6s
wxmGG1lIs2T0tsoG4/Kqw253hHR86wc8TTmLq02JMSrCYb2PAUmTQjOc5ReL2P4N
wxRGYGOV9NRCGev4+o45ly5/buhifznFljsObJvp/bPRnKkRrCM4YettGKunJWLH
g6fVSFscPdVt6SVLfh0niT6KPbYhYeDynCUbuNtCIebnWpT6UcJU01Bdp6yQ5o14
pcHhk6+gAq/a3/TfpMsMSje+iaUQTf4pBnYWVhAoOwWxp3TZ224N6cnTCdVYUdfq
Jps2cUSYOgJ/OqxE7dlF0p0QJzBXZDg4DO4lx2vT86RHF6lO1MSheQIDAQABoC0w
KwYJKoZIhvcNAQkOMR4wHDAaBgNVHREEEzARgg9wa2NzLmRvbWFpbi5jb20wDQYJ
KoZIhvcNAQELBQADggEBAAZMeiVH9HT9Ghn3GC7+83NIQSEI97vgo9tMPTysujSV
UaPBJYbVpSeE9B1fYVvcPuyHrV9/tQHunpbjJU5ZEsqfE8LEq2AcxKfPNSpWxl3F
dyRigSQc/GN3QDNZsSI1VKTjiYZ5yZhhSmZqL3S5BMY43tu5a3IV3hDuDk8GRovg
+gEWf2/z2NluqEwMuER0h7ltTX27qkk+d443EtSrwN50hSnqD6fMvfjcwbYq2Sw8
tOjt5lkQREgw2GFjObMr2w1IZXPy7bHDZYixUVZw4sizztHSuKK9yN5zPh6rzLAZ
jwPri7o275SdaSxZ7CmawSaeL0S33NJfHGAy5IeXngM=
-----END CERTIFICATE REQUEST-----
```

copy paste theCSR into files called 

* `ca_cratchpad/certs/softhsm-server.csr`
* `ca_cratchpad/certs/softhsm-client.csr`
  (yes, we're going to use the same csr for both end just to test for simplicity)

Generate Certs
```bash
cd ca_scratchpad
export NAME=softhsm-server
export SAN=DNS:pkcs.domain.com
openssl ca \
    -config tls-ca.conf \
    -in certs/$NAME.csr \
    -out certs/$NAME.crt \
    -extensions server_ext


export NAME=softhsm-client
export SAN=DNS:pkcs.domain.com

openssl ca \
    -config tls-ca.conf \
    -in certs/$NAME.csr \
    -out certs/$NAME.crt \
    -policy extern_pol \
    -extensions client_ext
```

You can also directly use openssl to create the CSR and then embed the cert into the HSM:

```bash
export PKCS11_PRIVATE_KEY="pkcs11:model=SoftHSM%20v2;manufacturer=SoftHSM%20project;serial=11819f2dd1b9d639;token=token1;type=private;object=keylabel1?pin-value=mynewpin"
export PKCS11_PUBLIC_KEY="pkcs11:model=SoftHSM%20v2;manufacturer=SoftHSM%20project;serial=11819f2dd1b9d639;token=token1;type=public;object=keylabel1?pin-value=mynewpin"

openssl req -new -x509 -config tls-ca.conf -extensions server_ext -engine pkcs11 -keyform engine -key "$PKCS11_PRIVATE_KEY" -outform pem -out certs/$NAME.crt
```

then after install, you should see two objects (public key, and a cert)

```bash
$ openssl x509 -in certs/$NAME.crt -out certs/$NAME.der -outform DER
$ pkcs11-tool --module [PATH]/libsofthsm2.so -l --id 4142 --label keylabel1 -y cert -w certs/$NAME.der --pin mynewpin

$ pkcs11-tool --module [PATH]/libsofthsm2.so  --list-objects
    Using slot 0 with a present token (0x451f46c3)
    Certificate Object; type = X.509 cert
      label:      keylabel1
      subject:    DN: C=US, ST=California, L=Mountain View, O=Google, OU=Enterprise, CN=pkcs.domain.com
      ID:         4142
    Public Key Object; RSA 2048 bits
      label:      keylabel1
      ID:         4142
      Usage:      encrypt, verify, wrap
      Access:     local
```
## Run the sample application

Before runnining the application, in 'main.go' update the project ID with your own project:
```go
  _, err = service.Projects.Topics.List("projects/[YOUR_PROJECT_ID]").Do()

```
And then:

```bash
go run .
```