res = 0.0

with open("results2.csv","r") as f:
    for line in f:
        relevant = float(line.strip().split(",")[-1])
        res += relevant
print(f"It took {res} seconds total")
