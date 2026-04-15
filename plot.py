import pandas as pd
import matplotlib.pyplot as plt

# Load CSV (no header)
df = pd.read_csv("resultsSemiLarge.csv", header=None)

# Last column = time
df["time"] = df.iloc[:, -1] * 1000

# Sum of all columns except last = resource count
df["resources"] = df.iloc[:, :-1].sum(axis=1)

# Define operation cycle
ops = ["add", "delete", "readd", "modify"]

# Assign operation labels cyclically
df["operation"] = [ops[i % 4] for i in range(len(df))]

# Plot
plt.figure(figsize=(10, 6))

for op in ops:
    subset = df[df["operation"] == op]
    plt.plot(subset["resources"], subset["time"], marker='o', label=op)

plt.xlabel("Total Resources")
plt.ylabel("Time (ms)")
plt.title("Operation Time vs Resource Count")
plt.legend()
plt.grid(True)

plt.savefig("plotsmlrg.png", dpi=400, bbox_inches="tight")
print("Plot saved as plotsmlrg.png")