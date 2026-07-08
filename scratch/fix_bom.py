import os
import glob
import codecs

paths = glob.glob(r'C:\Users\Admin\Desktop\*\1_تشغيل_الكل.*')

for path in paths:
    with codecs.open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    with codecs.open(path, 'w', encoding='utf-8-sig') as f:
        f.write(content)
        
paths2 = glob.glob(r'C:\Users\Admin\Desktop\*\2_إيقاف_الكل.*')
for path in paths2:
    with codecs.open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    with codecs.open(path, 'w', encoding='utf-8-sig') as f:
        f.write(content)
