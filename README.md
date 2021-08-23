# resumeio2pdf

This program downloads resumes from [resume.io](https://resume.io/) and saves them in PDF format (including links).

## Usage

```bash
resumeio2pdf [options] [ID or URL]
```

Options:
*  `-pdf` (string)  name of pdf file (default: `SecureID` + `.pdf`)
*  `-sid` (string) SecureID of resume
*  `-url` (string) link to resume of the format: https://resume.io/r/SecureID
*  `-verbose` show detail information
*  `-version` show version
*  `-y`	overwrite PDF file