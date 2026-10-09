"""Generate scene audio and sentence timings with official local Kokoro weights."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import sys

parser = argparse.ArgumentParser()
parser.add_argument('--output', type=Path, required=True)
parser.add_argument('--voice', default='bf_emma')
args = parser.parse_args()
args.output.mkdir(parents=True, exist_ok=True)
os.environ.setdefault('HF_HOME', str(args.output / 'model-cache'))
# spaCy's model installer must target this isolated interpreter.
os.environ['VIRTUAL_ENV'] = sys.prefix
os.environ['PATH'] = str(Path(sys.executable).parent) + os.pathsep + os.environ.get('PATH', '')

import numpy as np
import soundfile as sf
import torch
import espeakng_loader
from phonemizer.backend.espeak.wrapper import EspeakWrapper
from kokoro import KPipeline

EspeakWrapper.set_library(espeakng_loader.get_library_path())
os.environ['ESPEAK_DATA_PATH'] = espeakng_loader.get_data_path()
torch.set_num_threads(4)
scenes = json.loads(Path(__file__).with_name('storyboard.json').read_text())
pipeline = KPipeline(lang_code=args.voice[0], repo_id='hexgrad/Kokoro-82M', device='cpu')
timings = []
for scene in scenes:
    filename = args.output / (scene['id'] + '.wav')
    timing_file = args.output / (scene['id'] + '-timing.json')
    text_hash = hashlib.sha256(scene['narration'].encode()).hexdigest()
    if filename.exists() and timing_file.exists():
        cached = json.loads(timing_file.read_text())
        if cached.get('text_sha256') == text_hash and cached.get('voice') == args.voice:
            timings.append(cached)
            print('Cached', scene['id'], flush=True)
            continue
    samples = [np.zeros(4800, dtype=np.float32)]
    cursor = 0.2
    captions = []
    for sentence in re.split(r'(?<=[.!?])\s+', scene['narration']):
        parts = [audio.numpy() for _, _, audio in pipeline(sentence, voice=args.voice, speed=1.0)]
        if not parts:
            raise RuntimeError('No Kokoro audio for ' + sentence)
        audio = np.concatenate(parts)
        captions.append({'start': round(cursor, 3), 'end': round(cursor + len(audio)/24000, 3), 'text': sentence})
        samples.extend([audio, np.zeros(3600, dtype=np.float32)])
        cursor += len(audio)/24000 + 0.15
    samples.append(np.zeros(14400, dtype=np.float32))
    audio = np.concatenate(samples)
    sf.write(filename, audio, 24000, subtype='PCM_16')
    timing = {'id': scene['id'], 'duration': len(audio)/24000, 'voice': args.voice, 'text_sha256': text_hash, 'captions': captions}
    timing_file.write_text(json.dumps(timing, indent=2)+'\n')
    timings.append(timing)
    print(scene['id'], round(timing['duration'], 2), 'seconds', flush=True)
(args.output / 'timings.json').write_text(json.dumps(timings, indent=2)+'\n')
print('Total narration:', round(sum(t['duration'] for t in timings),2), 'seconds', flush=True)
