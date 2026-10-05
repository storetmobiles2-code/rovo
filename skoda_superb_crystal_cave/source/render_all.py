"""Parallel frame renderer.  usage: python render_all.py <scale> <outdir> [f0 f1] [workers]"""
import sys, os, time, multiprocessing as mp
import fx

def work(args):
    scale, outdir, a, b = args
    fx.set_scale(scale)
    import shots
    for f in range(a, b):
        shots.save_frame(f, outdir)
    return (a, b)

if __name__ == "__main__":
    scale = float(sys.argv[1]); outdir = sys.argv[2]
    f0 = int(sys.argv[3]) if len(sys.argv) > 3 else 0
    f1 = int(sys.argv[4]) if len(sys.argv) > 4 else 420
    nw = int(sys.argv[5]) if len(sys.argv) > 5 else 4
    os.makedirs(outdir, exist_ok=True)
    chunk = 8
    jobs = [(scale, outdir, a, min(a + chunk, f1)) for a in range(f0, f1, chunk)]
    t = time.time(); done = 0
    with mp.Pool(nw) as p:
        for a, b in p.imap_unordered(work, jobs):
            done += b - a
            print(f"{done}/{f1-f0} frames  {time.time()-t:.0f}s", flush=True)
