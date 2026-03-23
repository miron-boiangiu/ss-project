# Medical Images Folder

Acest folder conține imaginile cu fișele medicale care urmează să fie procesate.

## Utilizare

1. **Adaugă imaginile** - Pune toate fișele medicale scanate în acest folder
2. **Rulează scriptul** - Folosește scriptul de upload pentru a le trimite automat:
   ```bash
   python3 scripts/upload_folder.py medical-images
   ```

## Format suportat
- PNG (.png)
- JPEG (.jpg, .jpeg)

## Notă
Acest folder este ignorat de Git (este în `.gitignore`) pentru a nu include documente medicale sensibile în repository.
