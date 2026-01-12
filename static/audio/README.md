# Audio Files for SpellingClash

This directory contains audio pronunciations for spelling words.

## File Format

Audio files should be in MP3 format and named to match the `audio_filename` field in the database.

## Generating Audio Files

You can generate audio files using:

### Option 1: Text-to-Speech APIs
- **Google Cloud Text-to-Speech**: High quality, natural voices
- **Amazon Polly**: Good quality, various voices
- **Microsoft Azure Speech**: Multiple languages and voices

### Option 2: Manual Recording
Record pronunciations using a clear voice and save as MP3 files.

### Option 3: Text-to-Speech Tools
Use command-line tools like `say` on macOS or `espeak` on Linux:

```bash
# macOS example (requires converting to MP3 with ffmpeg)
say -o word.aiff "example"
ffmpeg -i word.aiff -acodec libmp3lame word.mp3
```

## Naming Convention

Audio files should be named descriptively, for example:
- `word_cat.mp3`
- `word_beautiful.mp3`
- `word_accommodate.mp3`

## Database Integration

When adding words to spelling lists, set the `audio_filename` field to match the audio file name:

```sql
UPDATE words SET audio_filename = 'word_cat.mp3' WHERE word_text = 'cat';
```

## Testing

1. Place audio files in this directory
2. Update the word's `audio_filename` in the database
3. Start a practice session - the audio will play automatically
4. Click "🔊 Play Word Again" to replay the audio

## Example Python Script to Generate Audio

```python
from gtts import gTTS
import os

words = ['cat', 'dog', 'house', 'beautiful', 'accommodate']

for word in words:
    tts = gTTS(text=word, lang='en', slow=False)
    tts.save(f'word_{word}.mp3')
    print(f'Generated: word_{word}.mp3')
```

Install: `pip install gtts`
