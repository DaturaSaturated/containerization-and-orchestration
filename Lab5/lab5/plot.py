import csv, datetime
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
times, codes = [], []
with open('load.csv') as f:
    for row in csv.reader(f):
        times.append(datetime.datetime.fromtimestamp(float(row[0]))); codes.append(int(row[1]))
kills = []
with open('chaos.log') as f:
    for line in f:
        kills.append(datetime.datetime.fromtimestamp(float(line.split()[0])))
fig, ax = plt.subplots(figsize=(12,4))
ax.plot(times, codes, label='HTTP code', color='tab:green')
for k in kills: ax.axvline(k, color='tab:red', alpha=0.3, linewidth=0.8)
ax.set_ylim(0,600); ax.set_yticks([0,200,500]); ax.set_yticklabels(['timeout','200 OK','error'])
ax.set_title('Continuity: red = pod killed, green stays 200'); ax.legend()
plt.tight_layout(); plt.savefig('continuity.png', dpi=120)
print('non-200 count:', sum(1 for c in codes if c != 200))
