"""Render designed API walkthrough frames, timed captions and a narrated MP4.

This is deliberately not a browser recorder. Only bundled sample API data is shown.
"""
import argparse
import json
import math
from pathlib import Path
import subprocess
import wave
from PIL import Image, ImageDraw, ImageFont
import imageio_ffmpeg

parser = argparse.ArgumentParser()
parser.add_argument('--audio', type=Path, required=True)
parser.add_argument('--frames-only', action='store_true')
parser.add_argument('--output', type=Path, required=True)
parser.add_argument('--font-dir', type=Path, default=Path('/System/Library/Fonts/Supplemental'))
args = parser.parse_args()
args.audio = args.audio.resolve()
args.output = args.output.resolve()
args.output.mkdir(parents=True, exist_ok=True)
work = args.audio.parent / 'render'
work.mkdir(exist_ok=True)
HERE = Path(__file__).parent
scenes = json.loads((HERE/'storyboard.json').read_text())
capture = json.loads((HERE/'capture.json').read_text())
timings = [] if args.frames_only else json.loads((args.audio/'timings.json').read_text())
ffmpeg = imageio_ffmpeg.get_ffmpeg_exe()
W,H,FPS = 1920,1080,24
BG,INK,MUTED,GREEN,PALE,WHITE,RED = '#edf3ef','#122b28','#50665f','#137654','#def0e5','#ffffff','#b23939'
FONT_NAMES={'normal':'Arial.ttf','bold':'Arial Bold.ttf','mono':'Andale Mono.ttf'}
def font(size, kind='normal'):
    return ImageFont.truetype(str(args.font_dir/FONT_NAMES[kind]), size)
def text(d, xy, value, size=32, color=INK, kind='normal'):
    d.text(xy,value,font=font(size,kind),fill=color)
def lines(value,width,size=32,kind='normal'):
    f=font(size,kind); result=[]
    for paragraph in value.split('\n'):
        line=''
        for word in paragraph.split():
            candidate=(line+' '+word).strip()
            if f.getlength(candidate)>width and line:
                result.append(line); line=word
            else: line=candidate
        result.append(line)
    return result

def paragraph(d,xy,value,width,size=32,color=MUTED,kind='normal',spacing=10):
    x,y=xy
    for line in lines(value,width,size,kind):
        text(d,(x,y),line,size,color,kind);y+=size+spacing
    return y

def card(d, box, fill=WHITE):
    d.rounded_rectangle(box,radius=24,fill=fill)

def pill(d,x,y,label,color=GREEN,fill=PALE):
    width=font(24,'bold').getlength(label)+38
    d.rounded_rectangle((x,y,x+width,y+46),radius=23,fill=fill)
    text(d,(x+19,y+9),label,24,color,'bold')

def base(scene,index):
    im=Image.new('RGB',(W,H),BG);d=ImageDraw.Draw(im)
    d.rounded_rectangle((70,52,122,104),radius=14,fill=GREEN)
    d.line([(84,80),(94,90),(111,68)],fill=WHITE,width=5)
    text(d,(139,60),'PolicyLens',36,INK,'bold')
    text(d,(1230,68),'WALKTHROUGH  /  RECORDED API RESULTS',23,MUTED)
    d.line((70,135,1850,135),fill='#c9d8d0',width=2)
    text(d,(74,167),scene['section'],24,GREEN,'bold')
    text(d,(70,211),scene['title'],74,INK,'bold')
    text(d,(1710,172),f'{index+1:02d} / 09',26,MUTED,'mono')
    return im,d

def metric(d,x,y,count,label,color=GREEN):
    text(d,(x,y),str(count),94,color,'bold')
    text(d,(x,y+107),label,27,MUTED)

def code(d,box,heading,value,highlights=()):
    card(d,box,'#152c28');x,y,x2,y2=box
    text(d,(x+30,y+26),heading,25,'#a2bfb1','mono')
    cy=y+81
    for line in value.splitlines():
        color='#93e0b7' if any(token in line for token in highlights) else '#e4f0e9'
        if font(27,'mono').getlength(line)>x2-x-60: raise ValueError('Code overflow: '+line)
        if cy+34>y2: raise ValueError('Code panel height overflow')
        text(d,(x+30,cy),line,27,color,'mono');cy+=39

def scene_image(scene,index):
    im,d=base(scene,index); sid=scene['id']
    if sid=='intro':
        card(d,(70,330,1060,843))
        pill(d,105,366,'PUBLIC KUBERNETES POLICIES')
        paragraph(d,(105,445),'From a policy rule to a\nconfiguration you can verify.',850,48,INK,'bold',17)
        paragraph(d,(105,619),'Source links, deterministic checks and evidence\nthat developers can inspect.',880,33,MUTED,spacing=15)
        text(d,(105,772),'Go  /  React  /  Kyverno  /  Evidence retrieval',27,GREEN,'bold')
        card(d,(1090,330,1850,843),INK)
        for n,(title,desc) in enumerate([('Explore','Read the rule and its original source'),('Check','Run the actual Kyverno engine'),('Explain','Inspect supporting passages')]):
            y=378+n*146
            d.ellipse((1125,y,1183,y+58),fill='#91d9b0')
            text(d,(1144,y+11),str(n+1),30,INK,'bold')
            text(d,(1210,y),title,38,WHITE,'bold')
            paragraph(d,(1210,y+51),desc,570,28,'#b5cec1')
    elif sid=='sources':
        for n,p in enumerate(capture['policies']):
            row,col=divmod(n,2);x=70+col*900;y=330+row*139
            card(d,(x,y,x+875,y+122))
            text(d,(x+26,y+21),f'{n+1:02d}',26,GREEN,'mono')
            text(d,(x+83,y+19),p['title'],33,INK,'bold')
            text(d,(x+83,y+69),p['category'],25,MUTED)
        card(d,(70,767,1850,843),INK)
        text(d,(103,790),'kyverno/policies   •   pinned revision '+capture['status']['provenance']['revision'][:12]+'   •   SHA-256 verified',29,WHITE,'mono')
    elif sid in ('failing','passing'):
        is_fail=sid=='failing';result=capture[sid]
        value=('image: docker.io/example/payment-api:latest\nsecurityContext:\n  privileged: true\n  runAsNonRoot: false\n  allowPrivilegeEscalation: true\n\n# Missing app.kubernetes.io/name label' if is_fail else 'labels:\n  app.kubernetes.io/name: payment-api\nimage: eu.foo.io/example/payment-api:1.0.0\nsecurityContext:\n  runAsNonRoot: true\n  privileged: false\n  allowPrivilegeEscalation: false')
        code(d,(70,330,1040,843),'POD MANIFEST / SELECTED FIELDS',value,('true' if is_fail else 'false','image:','app.kubernetes.io/name'))
        card(d,(1070,330,1850,843))
        pill(d,1104,360,'REAL KYVERNO '+result['engine'])
        metric(d,1110,425,result['summary']['pass'],'PASS')
        metric(d,1385,425,result['summary']['fail'],'FAIL',RED if is_fail else GREEN)
        text(d,(1110,606),'0 skipped  /  0 engine errors',29,MUTED,'mono')
        paragraph(d,(1110,670),'A deterministic engine result.\nThe model does not judge compliance.',675,35,INK,'bold')
        text(d,(1110,788),'Offline check of the selected six-rule pack',26,MUTED)
    elif sid=='missing_label':
        code(d,(70,330,1040,843),'CHANGE / CORRECTED POD', 'metadata:\n  name: payment-api\n- labels:\n-   app.kubernetes.io/name: payment-api\n\n# All other fields unchanged',('labels:','app.kubernetes.io/name'))
        card(d,(1070,330,1850,843));r=capture[sid]
        metric(d,1110,365,r['summary']['pass'],'PASS');metric(d,1385,365,r['summary']['fail'],'FAIL',RED)
        failed=[x for x in r['results'] if x['status']=='fail']
        assert len(failed)==1
        pill(d,1110,568,'ISOLATED REGRESSION',RED,'#f7e4e2')
        paragraph(d,(1110,643),failed[0]['title'],660,37,INK,'bold')
        paragraph(d,(1110,718),'Required field:\napp.kubernetes.io/name',680,29,MUTED,'mono')
    elif sid=='answer':
        card(d,(70,330,1120,843));pill(d,104,360,'SOURCE EXCERPTS / NO LLM GENERATION')
        paragraph(d,(105,439),capture['answer']['question'],950,43,INK,'bold')
        text(d,(105,546),'eu.foo.io/   or   bar.io/',42,GREEN,'mono')
        paragraph(d,(105,627),'These are upstream example prefixes.\nConfigure your own production registries.',945,33,MUTED)
        text(d,(105,774),'Retrieved passages • inspectable citations',28,GREEN,'bold')
        card(d,(1150,330,1850,843),INK)
        text(d,(1188,367),'SUPPORTING EVIDENCE',25,'#91d9b0','bold')
        evidence=next(p for p in capture['answer']['evidence'] if 'Every regular' in p['text'])
        paragraph(d,(1188,439),evidence['text'],615,31,WHITE,spacing=13)
        text(d,(1188,727),'Original policy: Kyverno',29,'#91d9b0','bold')
        text(d,(1188,784),'Immutable source revision',25,'#b5cec1')
    elif sid=='abstention':
        card(d,(70,330,1120,843));pill(d,105,363,'OUTSIDE THE COLLECTION')
        paragraph(d,(105,444),capture[sid]['question'],940,45,INK,'bold')
        d.rounded_rectangle((105,566,1085,701),radius=17,fill=PALE)
        paragraph(d,(135,591),'No supporting evidence\nin this Kubernetes collection.',900,35,GREEN,'bold')
        text(d,(105,778),'Abstained • no evidence returned',30,MUTED)
        card(d,(1150,330,1850,843),INK)
        text(d,(1187,369),'AUTHORED RETRIEVAL EVALUATION',25,'#91d9b0','bold')
        text(d,(1187,445),'18 / 18',65,WHITE,'bold');text(d,(1187,527),'Policy questions: expected policy in top 3',26,'#b5cec1')
        text(d,(1187,599),'6 / 6',65,WHITE,'bold');text(d,(1187,681),'Unrelated questions: abstention',26,'#b5cec1')
        text(d,(1187,784),'Small test set; not general accuracy.',25,'#91d9b0')
    elif sid=='engineering':
        nodes=[('React + TypeScript','Policy explorer / Pod editor'),('Go REST API','SDK / CLI / shared contracts'),('Kyverno engine','Fixed policies / bounded execution')]
        for n,(title,desc) in enumerate(nodes):
            x=70+n*603;card(d,(x,330,x+575,523),INK if n==1 else WHITE)
            text(d,(x+28,370),title,36,WHITE if n==1 else INK,'bold')
            paragraph(d,(x+28,435),desc,510,28,'#b5cec1' if n==1 else MUTED)
            if n<2:text(d,(x+576,407),'→',36,GREEN,'bold')
        for n,(title,desc) in enumerate([('Reliability','Deadlines / concurrency bounds'),('Verification','API tests / actual engine fixtures'),('Observability','Request IDs / readiness / metrics')]):
            x=70+n*603;card(d,(x,556,x+575,843))
            pill(d,x+28,590,title.upper())
            paragraph(d,(x+28,674),desc,505,35,INK,'bold')
    elif sid=='closing':
        card(d,(70,330,1130,843),INK)
        paragraph(d,(110,373),'Go backend engineering.\nInspectable policy evidence.\nReproducible engine results.',970,46,WHITE,'bold',23)
        text(d,(110,671),'Free Cloudflare Tunnel preview',34,'#91d9b0','bold')
        text(d,(110,728),'Host + server + tunnel must remain online.',28,'#b5cec1')
        text(d,(110,786),'github.com/RenuBhati/policylens',27,'#91d9b0','mono')
        card(d,(1160,330,1850,843))
        text(d,(1197,373),'DEMO BOUNDARIES',26,GREEN,'bold')
        paragraph(d,(1197,440),'Six selected rules, checked offline.\n\nNo cluster certification.\nNo admission enforcement.\nNo real-model quality claim.',610,33,INK,spacing=12)
        text(d,(1197,782),'Narration: Kokoro / Emma',26,MUTED)
    return im

# Captions are burned in and separately exported. They describe speech verbatim.
def captioned(im,value):
    im=im.copy();d=ImageDraw.Draw(im)
    d.rounded_rectangle((70,899,1850,1040),radius=20,fill=INK)
    ls=lines(value,1680,32)
    if len(ls)>3:raise ValueError('Caption overflow: '+value)
    y=969-len(ls)*20
    for line in ls:
        width=font(32).getlength(line)
        text(d,((W-width)/2,y),line,32,WHITE);y+=41
    return im

def timestamp(t):
    ms=round(t*1000);s,ms=divmod(ms,1000);m,s=divmod(s,60);h,m=divmod(m,60)
    return f'{h:02d}:{m:02d}:{s:02d},{ms:03d}'

if args.frames_only:
    contact=Image.new('RGB',(1920,1080),BG)
    for i,s in enumerate(scenes):
        im=scene_image(s,i)
        im.save(work/(s['id']+'.png'))
        contact.paste(im.resize((640,360)),((i%3)*640,(i//3)*360))
    contact.save(args.output/'PolicyLens_Storyboard.jpg',quality=92)
    print('Rendered nine preview frames',flush=True)
    raise SystemExit(0)

srt=[];total=0;clips=[];thumbs=[];all_audio=[];chapters=[]
for i,(scene,timing) in enumerate(zip(scenes,timings)):
    assert scene['id']==timing['id']
    im=scene_image(scene,i)
    im.save(work/(scene['id']+'.png'))
    thumbs.append(im.copy().resize((640,360)))
    duration=math.ceil(timing['duration']*FPS)/FPS
    chapters.append({'start':round(total,3),'title':scene['title'],'id':scene['id']})
    frame_list=[]
    first=work/(scene['id']+'-empty.png');captioned(im,'').save(first)
    frame_list.append((first,timing['captions'][0]['start']))
    for j,c in enumerate(timing['captions']):
        path=work/f'{scene["id"]}-caption-{j}.png';captioned(im,c['text']).save(path)
        next_start=timing['captions'][j+1]['start'] if j+1<len(timing['captions']) else duration
        frame_list.append((path,next_start-c['start']))
        srt.append(f'{len(srt)+1}\n{timestamp(total+c["start"])} --> {timestamp(total+c["end"])}\n{c["text"]}\n')
    concat=work/(scene['id']+'-frames.txt')
    concat.write_text(''.join(f"file '{str(p).replace(chr(39), chr(39)+chr(92)+chr(39)+chr(39))}'\nduration {d:.6f}\n" for p,d in frame_list)+f"file '{frame_list[-1][0]}'\n")
    clip=work/(scene['id']+'.mp4');clips.append(clip)
    log=work/(scene['id']+'-ffmpeg.log')
    cmd=[ffmpeg,'-y','-f','concat','-safe','0','-i',str(concat),'-i',str(args.audio/(scene['id']+'.wav')),'-vf',f'fps={FPS},fade=t=in:st=0:d=0.2,fade=t=out:st={duration-0.2:.6f}:d=0.2','-af','apad','-t',f'{duration:.6f}','-c:v','libx264','-preset','fast','-crf','20','-pix_fmt','yuv420p','-c:a','aac','-b:a','160k','-ar','24000','-movflags','+faststart',str(clip)]
    with log.open('w') as f:subprocess.run(cmd,stdout=f,stderr=f,check=True)
    with wave.open(str(args.audio/(scene['id']+'.wav')),'rb') as f:
        assert f.getframerate()==24000 and f.getnchannels()==1 and f.getsampwidth()==2
        raw=f.readframes(f.getnframes())
        all_audio.append(raw+b'\x00'*max(0,(round(duration*24000)-len(raw)//2)*2))
    total+=duration
    print('Rendered',scene['id'],round(duration,2),'seconds',flush=True)

join=work/'clips.txt';join.write_text(''.join(f"file '{p}'\n" for p in clips))
video=args.output/'PolicyLens_Demo.mp4'
with (work/'final-ffmpeg.log').open('w') as f:
    subprocess.run([ffmpeg,'-y','-f','concat','-safe','0','-i',str(join),'-c:v','copy','-c:a','aac','-b:a','160k','-af','loudnorm=I=-16:TP=-1.5:LRA=11','-ar','48000','-movflags','+faststart',str(video)],stdout=f,stderr=f,check=True)
(args.output/'PolicyLens_Demo.srt').write_text('\n'.join(srt))
raw_narration = work/'narration-raw.wav'
with wave.open(str(raw_narration),'wb') as f:
    f.setnchannels(1);f.setsampwidth(2);f.setframerate(24000);f.writeframes(b''.join(all_audio))
with (work/'narration-ffmpeg.log').open('w') as f:
    subprocess.run([ffmpeg,'-y','-i',str(raw_narration),'-af','loudnorm=I=-16:TP=-1.5:LRA=11','-ar','24000','-c:a','pcm_s16le',str(args.output/'PolicyLens_Narration.wav')],stdout=f,stderr=f,check=True)
contact=Image.new('RGB',(1920,1080),BG)
for i,im in enumerate(thumbs):contact.paste(im,((i%3)*640,(i//3)*360))
contact.save(args.output/'PolicyLens_Storyboard.jpg',quality=92)
poster=captioned(scene_image(scenes[0],0),'A narrated walkthrough with real API results.').resize((1280,720))
poster.save(args.output/'PolicyLens_Demo_Poster.jpg',quality=95)
metadata={'duration_seconds':total,'resolution':'1920x1080','fps':FPS,'voice':'Kokoro bf_emma (local official model)','format':'Designed walkthrough using recorded API results; not a screen recording','captured_at':capture['captured_at'],'chapters':chapters}
(args.output/'PolicyLens_Demo_Info.json').write_text(json.dumps(metadata,indent=2)+'\n')
transcript='# PolicyLens demo transcript\n\nBritish English narration: Kokoro Emma (`bf_emma`), generated locally with the official `hexgrad/Kokoro-82M` model. The requested hosted Space disables public API access; its model is used locally.\n\nDesigned walkthrough with actual captured API results; this is not a browser screen recording. No real LLM-generated answers are shown. Captions are burned into the video and supplied as a separate SRT.\n\n'
for c,s in zip(chapters,scenes):transcript+=f'## {timestamp(c["start"]).split(",")[0]} — {s["title"]}\n\n{s["narration"]}\n\n'
transcript+='Sources: [Requested Kokoro Space](https://huggingface.co/spaces/hexgrad/Kokoro-TTS), [official model](https://huggingface.co/hexgrad/Kokoro-82M), [PolicyLens repository](https://github.com/RenuBhati/policylens).\n'
(args.output/'PolicyLens_Demo_Transcript.md').write_text(transcript)
print('Saved',video,'duration',round(total,2),'seconds',flush=True)
