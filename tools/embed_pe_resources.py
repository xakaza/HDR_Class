#!/usr/bin/env python3
import argparse
import os
import struct
import subprocess
import tempfile
import shutil
from pathlib import Path

LANG_EN_US = 1033
RT_ICON = 3
RT_GROUP_ICON = 14
RT_VERSION = 16
RT_MANIFEST = 24


def align4(n):
    return (n + 3) & ~3


def u16z(s):
    return s.encode('utf-16le') + b'\x00\x00'


def version_block(key, value=b'', children=(), wtype=0, value_length=None):
    body = bytearray()
    body += b'\x00' * 6
    body += u16z(key)
    while len(body) % 4:
        body += b'\x00'
    body += value
    while len(body) % 4:
        body += b'\x00'
    for child in children:
        body += child
        while len(body) % 4:
            body += b'\x00'
    if value_length is None:
        value_length = len(value) if wtype == 0 else len(value) // 2
    struct.pack_into('<HHH', body, 0, len(body), value_length, wtype)
    return bytes(body)


def make_string(key, value):
    vb = u16z(value)
    return version_block(key, vb, (), 1, len(value) + 1)


def make_version_info(version=(1, 0, 3, 0)):
    a,b,c,d = version
    ms = ((a & 0xffff) << 16) | (b & 0xffff)
    ls = ((c & 0xffff) << 16) | (d & 0xffff)
    ffi = struct.pack(
        '<13I',
        0xFEEF04BD, 0x00010000,
        ms, ls, ms, ls,
        0x0000003F, 0,
        0x00040004,  # VOS_NT_WINDOWS32
        0x00000001,  # VFT_APP
        0, 0, 0
    )
    strings = [
        make_string('CompanyName', 'HDR Studio Project'),
        make_string('FileDescription', 'HDR Studio - HDR/SRC Packer & Unpacker'),
        make_string('FileVersion', f'{a}.{b}.{c}.{d}'),
        make_string('InternalName', 'HDRStudio'),
        make_string('OriginalFilename', 'HDRStudio.exe'),
        make_string('ProductName', 'HDR Studio'),
        make_string('ProductVersion', f'{a}.{b}.{c}.{d}'),
    ]
    table = version_block('040904E4', b'', strings, 1, 0)
    sfi = version_block('StringFileInfo', b'', [table], 1, 0)
    translation = struct.pack('<HH', 0x0409, 0x04E4)
    var = version_block('Translation', translation, (), 0, len(translation))
    vfi = version_block('VarFileInfo', b'', [var], 1, 0)
    return version_block('VS_VERSION_INFO', ffi, [sfi, vfi], 0, len(ffi))


def make_manifest():
    return b'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">\n  <assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="HDRStudio" type="win32"/>\n  <description>HDR Studio</description>\n  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">\n    <security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security>\n  </trustInfo>\n  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1"><application>\n    <supportedOS Id="{4f476546-937d-4f4f-99e1-8d02b47d73e5}"/>\n    <supportedOS Id="{1e2b8b7d-ef0b-4e72-8e2b-3a1e22f66063}"/>\n    <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>\n  </application></compatibility>\n  <application xmlns="urn:schemas-microsoft-com:asm.v3"><windowsSettings>\n    <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>\n    <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">PerMonitorV2</dpiAwareness>\n    <longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware>\n  </windowsSettings></application>\n  <dependency><dependentAssembly>\n    <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>\n  </dependentAssembly></dependency>\n</assembly>\n'''


def read_best_ico_image(path):
    b = Path(path).read_bytes()
    if len(b) < 6:
        raise ValueError('invalid ico')
    reserved, typ, count = struct.unpack_from('<HHH', b, 0)
    if reserved != 0 or typ != 1 or count < 1:
        raise ValueError('not an icon file')
    entries=[]
    for i in range(count):
        off=6+i*16
        w,h,cc,res,planes,bpp,size,dataoff=struct.unpack_from('<BBBBHHII',b,off)
        ww=256 if w==0 else w
        hh=256 if h==0 else h
        if dataoff+size>len(b):
            continue
        entries.append((ww*hh,bpp,ww,hh,planes,size,dataoff,b[dataoff:dataoff+size]))
    if not entries:
        raise ValueError('no valid ico images')
    entries.sort(reverse=True)
    _, bpp, w, h, planes, size, dataoff, data = entries[0]
    return data, w, h, planes or 1, bpp or 32


def pack_dir(num_ids):
    return struct.pack('<IIHHHH',0,0,0,0,0,num_ids)


def pack_entry(idv, off, isdir):
    return struct.pack('<II', idv, off | (0x80000000 if isdir else 0))


def build_resource(icon_data, w, h, planes, bpp, version_data, manifest_data, base_rva=0):
    # One ID and one language per resource type. The icon group references icon ID 1.
    group_data = struct.pack('<HHH',0,1,1)
    bw=0 if w>=256 else w
    bh=0 if h>=256 else h
    group_data += struct.pack('<BBBBHHIH',bw,bh,0,0,planes,bpp,len(icon_data),1)
    payloads = [
        (RT_ICON, 1, icon_data),
        (RT_GROUP_ICON, 1, group_data),
        (RT_VERSION, 1, version_data),
        (RT_MANIFEST, 1, manifest_data),
    ]
    n=len(payloads)
    root_off=0
    cursor=16+n*8
    type_dirs=[]
    id_dirs=[]
    data_entries=[]
    for _ in payloads:
        type_dirs.append(cursor); cursor += 24
        id_dirs.append(cursor); cursor += 24
        data_entries.append(cursor); cursor += 16
    cursor=align4(cursor)
    payload_offsets=[]
    for _,_,data in payloads:
        payload_offsets.append(cursor)
        cursor=align4(cursor+len(data))
    out=bytearray(cursor)
    out[root_off:root_off+16]=pack_dir(n)
    for i,(rtype,rid,data) in enumerate(payloads):
        out[16+i*8:24+i*8]=pack_entry(rtype,type_dirs[i],True)
        td=type_dirs[i]
        out[td:td+16]=pack_dir(1)
        out[td+16:td+24]=pack_entry(rid,id_dirs[i],True)
        ld=id_dirs[i]
        out[ld:ld+16]=pack_dir(1)
        out[ld+16:ld+24]=pack_entry(LANG_EN_US,data_entries[i],False)
        de=data_entries[i]
        po=payload_offsets[i]
        struct.pack_into('<IIII',out,de,base_rva+po,len(data),0,0)
        out[po:po+len(data)]=data
    return bytes(out), data_entries, payload_offsets


def parse_pe(data):
    if data[:2] != b'MZ': raise ValueError('not PE')
    pe=struct.unpack_from('<I',data,0x3c)[0]
    if data[pe:pe+4] != b'PE\0\0': raise ValueError('bad PE')
    coff=pe+4
    machine,nsec,_,_,_,optsz,_=struct.unpack_from('<HHIIIHH',data,coff)
    opt=coff+20
    magic=struct.unpack_from('<H',data,opt)[0]
    dd_off=opt+(112 if magic==0x20b else 96 if magic==0x10b else (_ for _ in ()).throw(ValueError('bad optional header')))
    image_base = struct.unpack_from('<Q' if magic==0x20b else '<I', data, opt+24 if magic==0x20b else opt+28)[0]
    section_alignment = struct.unpack_from('<I', data, opt+32)[0]
    sh=opt+optsz
    secs=[]
    for i in range(nsec):
        o=sh+i*40
        name=data[o:o+8].split(b'\0',1)[0].decode('ascii','ignore')
        vsize,vaddr,rsize,rptr=struct.unpack_from('<IIII',data,o+8)
        secs.append((name,vsize,vaddr,rsize,rptr,o))
    return pe,opt,dd_off,secs,image_base,section_alignment


def embed(exe, ico, out, version):
    exe=Path(exe); out=Path(out)
    icon_data,w,h,planes,bpp=read_best_ico_image(ico)
    version_data=make_version_info(version)
    manifest_data=make_manifest()
    rsrc,data_entries,payload_offsets=build_resource(icon_data,w,h,planes,bpp,version_data,manifest_data,0)
    original=exe.read_bytes()
    _,_,_,secs0,image_base,section_alignment=parse_pe(original)
    max_vma=image_base
    for _name,vsize,vaddr,rsize,rptr,_o in secs0:
        max_vma=max(max_vma,image_base+vaddr+max(vsize,rsize))
    next_vma=(max_vma+section_alignment-1)//section_alignment*section_alignment
    with tempfile.TemporaryDirectory() as td:
        td=Path(td)
        rb=td/'rsrc.bin'; rb.write_bytes(rsrc)
        tmp=td/'withrsrc.exe'
        objcopy=os.environ.get('OBJCOPY') or shutil.which('objcopy') or shutil.which('llvm-objcopy')
        if not objcopy:
            raise RuntimeError('objcopy or llvm-objcopy was not found')
        subprocess.check_call([objcopy,'--add-section',f'.rsrc={rb}','--set-section-flags','.rsrc=alloc,load,readonly,data','--change-section-vma',f'.rsrc=0x{next_vma:x}',str(exe),str(tmp)])
        d=bytearray(tmp.read_bytes())
        _,_,dd,secs,_,_=parse_pe(d)
        sec=next((x for x in secs if x[0]=='.rsrc'),None)
        if not sec: raise RuntimeError('objcopy did not add .rsrc')
        _,_,rva,_,rptr,_=sec
        for de,po in zip(data_entries,payload_offsets):
            struct.pack_into('<I',d,rptr+de,rva+po)
        struct.pack_into('<II',d,dd+2*8,rva,len(rsrc))
        out.write_bytes(d)
    print(f'embedded icon + version + manifest -> {out}')


def parse_version(s):
    parts=[int(x) for x in s.split('.')]
    if len(parts)>4: raise ValueError('version must have at most 4 numeric parts')
    parts += [0]*(4-len(parts))
    if any(x<0 or x>65535 for x in parts): raise ValueError('version component out of range')
    return tuple(parts)


if __name__=='__main__':
    ap=argparse.ArgumentParser()
    ap.add_argument('input_exe')
    ap.add_argument('icon_ico')
    ap.add_argument('output_exe')
    ap.add_argument('--version',default='1.0.3.0')
    a=ap.parse_args()
    embed(a.input_exe,a.icon_ico,a.output_exe,parse_version(a.version))
