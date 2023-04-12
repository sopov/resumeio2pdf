# Docker example

## Usage

Build image:

```bash
docker build -t resumeio2pdf .
```

Run image:

```bash
docker run -v {HOST_ABSOLUTE_FOLDER_PATH}:/pdfs -e SecureID={YOUR_SECURE_ID} -e FilePath=/pdfs/test.pdf resumeio2pdf
```

Run image (docker-compose):

```yml
version: "3"

services:
  resumeio2pdf:
    build: .
    environment:
      - SecureID={YOUR_SECURE_ID}
      - Filename=/pdfs/test.pdf
    volumes:
      - ./pdfs:/pdfs
```

To obtain the secure id: https://resume.io/r/SecureID