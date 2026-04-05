# miniblue Pulumi Python example

Start miniblue first: `./bin/miniblue`

```bash
cd examples/pulumi-python
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
pulumi login --local
pulumi stack init dev
pulumi up --yes --skip-preview
pulumi destroy --yes --skip-preview
```
