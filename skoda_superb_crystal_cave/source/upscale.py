import torch, numpy as np, cv2, sys, time
from spandrel import ModelLoader
torch.set_num_threads(4)
m=ModelLoader().load_from_file('models/RealESRGAN_x4plus.pth').eval()
U='/root/.claude/uploads/61fee336-d9f4-52f5-96b4-01e0ca924fb0/'
src={'rear34':'2639c78c-image.jpg','side':'09e46400-image.jpg','front':'71696533-image.jpg'}
def run(img,tile=160,pad=16):
    h,w,_=img.shape; out=np.zeros((h*4,w*4,3),np.float32)
    for y in range(0,h,tile):
        for x in range(0,w,tile):
            y0,x0=max(y-pad,0),max(x-pad,0); y1,x1=min(y+tile+pad,h),min(x+tile+pad,w)
            t=torch.from_numpy(img[y0:y1,x0:x1]).permute(2,0,1)[None]
            with torch.no_grad(): o=m(t)[0].permute(1,2,0).clamp(0,1).numpy()
            oy,ox=(y-y0)*4,(x-x0)*4; th,tw=min(tile,h-y),min(tile,w-x)
            out[y*4:(y+th)*4,x*4:(x+tw)*4]=o[oy:oy+th*4,ox:ox+tw*4]
    return out
for k,f in src.items():
    t=time.time()
    bgr=cv2.imread(U+f); rgb=cv2.cvtColor(bgr,cv2.COLOR_BGR2RGB).astype(np.float32)/255
    up=run(rgb); up8=(up*255).astype(np.uint8)
    a=cv2.imread(f'cut/{k}.png',cv2.IMREAD_UNCHANGED)[:,:,3]
    a=cv2.resize(a,(up8.shape[1],up8.shape[0]),interpolation=cv2.INTER_CUBIC).astype(np.float32)/255
    a=cv2.GaussianBlur(a,(0,0),2.0); a=np.clip((a-0.5)*2.2+0.5,0,1)
    # shrink matte slightly to avoid halo
    a=cv2.erode((a*255).astype(np.uint8),np.ones((3,3),np.uint8)).astype(np.float32)/255
    a=cv2.GaussianBlur(a,(0,0),1.2)
    rgba=np.dstack([cv2.cvtColor(up8,cv2.COLOR_RGB2BGR),(a*255).astype(np.uint8)])
    cv2.imwrite(f'up/{k}_x4.png',rgba); print(k,rgba.shape,time.time()-t,flush=True)
