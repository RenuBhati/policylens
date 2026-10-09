"""Refresh the recorded demo data from the sample workspace only."""
import argparse
import datetime
import json
from pathlib import Path
import urllib.request

parser=argparse.ArgumentParser()
parser.add_argument('--origin',default='http://127.0.0.1:8080')
args=parser.parse_args()
def api(path,body=None):
    req=urllib.request.Request(args.origin.rstrip('/')+path,data=None if body is None else json.dumps(body).encode(),headers={'Content-Type':'application/json'})
    with urllib.request.urlopen(req,timeout=30) as response:
        return json.load(response)

data={'captured_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'policies':api('/api/policies'),'examples':api('/api/examples'),'status':api('/api/status')}
for name in ['failing','passing']:
    data[name]=api('/api/check',{'manifest':data['examples'][name]})
manifest=data['examples']['passing'].replace('  labels:\n    app.kubernetes.io/name: payment-api\n','')
data['missing_label']=api('/api/check',{'manifest':manifest})
for name,question in [('answer','Which image registries are allowed?'),('abstention','How much annual leave do I get?')]:
    data[name]={'question':question,**api('/api/ask',{'question':question})}
for name,expected in [('failing',(0,6)),('passing',(6,0)),('missing_label',(5,1))]:
    summary=data[name]['summary']
    if (summary['pass'],summary['fail'],summary['skipped'],summary['error'])!=(*expected,0,0):
        raise RuntimeError('Unexpected fixture result: '+name)
if data['status']['answer_mode']!='excerpts' or data['answer']['abstained'] or not data['abstention']['abstained']:
    raise RuntimeError('The walkthrough requires source-excerpt mode and the expected question results')
Path(__file__).with_name('capture.json').write_text(json.dumps(data,indent=2)+'\n')
print('Captured sample workspace results at',data['captured_at'])
