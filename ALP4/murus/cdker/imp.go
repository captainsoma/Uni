package cdker

// (c) Christian Maurer   v. 130227 - license see murus.go

//#include <stdlib.h>
//#include <stdio.h>
//#include <fcntl.h>
//#include <sys/ioctl.h>
//#include <linux/cdrom.h>
//#include <GL/gl.h>
//#include <GL/glx.h>
/*
unsigned char nTracks (int d)
{
  struct cdrom_tochdr h;
  if (ioctl (d, CDROMREADTOCHDR, &h) < 0) return 0;
  return h.cdth_trk1;
}

unsigned int startFrame (int d, unsigned char t)
{
  unsigned char n;
  struct cdrom_tocentry e;
  unsigned int f;
  n = nTracks (d);
  if (n == 0) return 1000 * 1000 * 1000;
  if (t >= n)
    e.cdte_track = CDROM_LEADOUT;
  else
    e.cdte_track = t + 1;
  e.cdte_format = CDROM_MSF;
  if (ioctl (d, CDROMREADTOCENTRY, &e) < 0) return 0;
  f = e.cdte_addr.msf.minute * CD_SECS;
  f += e.cdte_addr.msf.second;
  f *= CD_FRAMES;
  f += e.cdte_addr.msf.frame;
  return f;
}

int status (int d, unsigned int *f)
{
  *f = 0;
  struct cdrom_subchnl c;
  unsigned char s;
  c.cdsc_format = CDROM_MSF;
  if (ioctl (d, CDROMSUBCHNL, &c) < 0) {
    return 0;
  }
  switch (c.cdsc_audiostatus) {
  case CDROM_AUDIO_INVALID:
    s = 0; break;
  case CDROM_AUDIO_PLAY:
    s = 1; break;
  case CDROM_AUDIO_PAUSED:
    s = 2; break;
  case CDROM_AUDIO_COMPLETED:
    s = 3; break;
  case CDROM_AUDIO_ERROR:
    s = 4; break;
  case CDROM_AUDIO_NO_STATUS:
    s = 5; break;
  }
  *f = c.cdsc_absaddr.msf.minute * CD_SECS;
  *f += c.cdsc_absaddr.msf.second;
  *f *= CD_FRAMES;
  *f += c.cdsc_absaddr.msf.frame;
  return s;
}

int start (int d)
{
  if (ioctl (d, CDROMSTART) < 0) return 0;
  return 1;
}

void play (int d, unsigned int f, unsigned int f1)
{
  struct cdrom_msf p;
  p.cdmsf_min0 = (f / CD_FRAMES) / CD_SECS;
  p.cdmsf_sec0 = (f / CD_FRAMES) % CD_SECS;
  p.cdmsf_frame0 = f % CD_FRAMES;
  p.cdmsf_min1 = (f1 / CD_FRAMES) / CD_SECS;
  p.cdmsf_sec1 = (f1 / CD_FRAMES) % CD_SECS;
  p.cdmsf_frame1 = f1 % CD_FRAMES;
  ioctl (d, CDROMPLAYMSF, &p);
}

void volRead (int d, unsigned char *l, unsigned char *r)
{
  *l = 0; *r = 0;
  struct cdrom_volctrl v;
  if (ioctl (d, CDROMVOLREAD, &v) < 0) return;
  if (v.channel0 >= 0 && v.channel0 <= 255) { *l = v.channel0; }
  if (v.channel1 >= 0 && v.channel1 <= 255) { *r = v.channel1; }
}

void volCtrl (int d, unsigned char l, unsigned char r)
{
  struct cdrom_volctrl v;
  v.channel0 = l; v.channel1 = r;
  v.channel2 = 0; v.channel3 = 0;
  ioctl (d, CDROMVOLCTRL, &v);
}

void lockDoor (int d) { ioctl (d, CDROM_LOCKDOOR); }

void closeTray (int d) { ioctl (d, CDROMCLOSETRAY); }

void pause (int d) { ioctl (d, CDROMPAUSE); }

void resume (int d) { ioctl (d, CDROMRESUME); }

void eject (int d) { ioctl (d, CDROMEJECT); }

void stop (int d) { ioctl (d, CDROMSTOP); }
*/
import
  "C"
import (
  "os"
  "syscall"
  "murus/ker"
  "murus/env"
  "murus/clk"
)
const (invalid = iota
  playing; paused; completed; fault; nostatus
  nStat
)
const (
  frames = 75 // cdrom.h: CD_FRAMES
  offset = 2 * frames
)
var (
  cdfile *os.File
  cdd int
  startFrame []uint
  nTracks uint8
  volume_left, volume_right, balance uint8
  status int
  text [nStat]string
)


func cstatus (d int) (uint, int) {
//
  var f C.uint
  i:= C.status (C.int(d), &f)
  return uint(f), int(i)
}


func inTrack (f uint) uint8 {
//
  t:= nTracks
  if f >= startFrame[0] && f < startFrame[t] - offset {
    t --
    for f < startFrame[t] {
      t --
    }
  }
  return t
}


func ms (f uint) (uint, uint) {
//
  f += frames / 2 // rounding !
  f /= frames
  return f / 60, f % 60
}


func actTrack () uint8 {
//
  var f uint
  f, status = cstatus (cdd)
  m, s:= ms (f - offset)
  Time.Set (m / 60, m % 60, s)
  a:= inTrack (f)
  m, s = ms (f - startFrame[a])
  TrackTime.Set (m / 60, m % 60, s)
  return a
}


func string_ () string {
//
  return text[status]
}


func soundfile () *os.File {
//
  dev:= env.Par (1)
  if dev == "" { dev = "cdrom" }
  var e error
  cdfile, e = os.OpenFile ("/dev/" + dev, syscall.O_RDONLY | syscall.O_NONBLOCK, 0440)
  if e != nil {
    return nil
  }
  cdd = int(cdfile.Fd())
  C.closeTray (C.int(cdd))
  counter:= 0
  for { // anfangs dauert's manchmal 'ne Weile ...
    counter ++
    if counter > 30 {
      return nil
    }
    _, status = cstatus (cdd)
    if status == invalid {
      ker.Msleep (250 * 1000)
    } else {
      break
    }
  }
  nTracks = uint8(C.nTracks (C.int(cdd)))
  n1:= nTracks + 1
  startFrame = make ([]uint, n1)
  StartTime = make ([]*clk.Imp, nTracks)
  Length = make ([]*clk.Imp, nTracks)
  for t:= uint8(0); t <= nTracks; t++ {
    startFrame[t] = uint(C.startFrame (C.int(cdd), C.uchar(t)))
    if t < nTracks {
      StartTime[t] = clk.New()
      Length[t] = clk.New()
      m, s:= ms (startFrame[t] - offset)
      StartTime[t].Set (m / 60, m % 60, s)
    }
    if t > 0 {
      m, s:= ms (startFrame[t] - startFrame[t - 1] - offset)
      Length[t - 1].Set (m / 60, m % 60, s)
    }
  }
  m, s:= ms (startFrame[nTracks] - 2 * offset)
  TotalTime.Set (m / 60, m % 60, s)
  var l, r C.uchar
  C.volRead (C.int(cdd), &l, &r)
  volume_left, volume_right = uint8(l), uint8(r)
//  C.lockDoor (C.int(cdd))
  Ctrl (All, MaxVol / 3)
  return cdfile
}


func playTrack (t uint8) () {
//
  if t >= nTracks { return }
  if int(C.start (C.int(cdd))) == 0 { return }
  C.play (C.int(cdd), C.uint(startFrame[t]), C.uint(startFrame[nTracks] - offset))
}


func playTrack0 () {
//
  var f uint
  f, status = cstatus (cdd)
  playTrack (inTrack (f))
}


func playTrack1 (fwd bool) {
//
  var f uint
  f, status = cstatus (cdd)
  t:= inTrack (f)
  if fwd {
    if t + 1 < nTracks {
      playTrack (t + 1)
    }
  } else if t > 0 {
    playTrack (t - 1)
  }
}


func posTime1 (fwd bool, sec uint) () {
//
  if sec == 0 { return }
  var f uint
  f, status = cstatus (cdd)
  s:= f / frames
  if fwd {
    s += sec
  } else {
    if s >= sec {
      s -= sec
    } else {
      s = 0
    }
  }
  posTime (true, s)
}


func posTime (all bool, sec uint) {
//
  f:= frames * sec
  t:= uint8(0)
  if ! all {
    var f1 uint
    f1, status = cstatus (cdd)
    t = inTrack (f1)
  }
  f += startFrame[t]
  if f > startFrame[nTracks] { return }
  C.play (C.int(cdd), C.uint(f), C.uint(startFrame[nTracks] - offset))
}


func switch_ () {
//
  switch status { case playing:
    C.pause (C.int(cdd))
  case paused:
    C.resume (C.int(cdd))
  }
}


func term () {
//
  C.stop (C.int(cdd))
}


func term1 () {
//
  C.stop (C.int(cdd))
  C.lockDoor (C.int(cdd))
  C.eject (C.int(cdd))
}


func volume (c Controller) uint8 {
//
  l, r:= uint(volume_left), uint(volume_right)
  sum:= l + r
  switch c { case Left:
    return volume_left
  case Right:
    return volume_right
  case Balance:
    if volume_left == volume_right { return MaxVol / 2 }
    return uint8((r * uint(MaxVol)) / sum)
  }
  return uint8(sum / 2)
}


func ctrl1 (c Controller, lauter bool) {
//
  switch c { case Left:
    if lauter {
      if volume_left < MaxVol { volume_left ++ }
    } else {
      if volume_left > 0 { volume_left -- }
    }
  case Right:
    if lauter {
      if volume_right < MaxVol { volume_right ++ }
    } else {
      if volume_right > 0 { volume_right -- }
    }
  case Balance:
    if lauter {
      if volume_left < MaxVol { volume_left ++ }
      if volume_right > 0 { volume_right -- }
    } else {
      if volume_right < MaxVol { volume_right ++ }
      if volume_left > 0 { volume_left -- }
    }
  case All:
    if lauter {
      if volume_left < MaxVol { volume_left ++ }
      if volume_right < MaxVol { volume_right ++ }
    } else {
      if volume_left > 0 { volume_left -- }
      if volume_right > 0 { volume_right -- }
    }
  }
  C.volCtrl (C.int(cdd), C.uchar(volume_left), C.uchar(volume_right))
  balance = Volume (Balance)
}


func ctrl (c Controller, i uint8) {
//
  l, r, j:= uint(volume_left), uint(volume_right), uint(i)
  sum:= l + r
  mid:= sum / 2
  switch c {case Left:
    l = j
  case Right:
    r = j
  case Balance:
    if i == 0 {
      l, r = sum, 0
    } else if j == MaxVol {
      l, r = 0, sum
    } else {
      r = (sum * j) / MaxVol
      l = sum - r
    }
  case All:
    if l == r {
      l, r = j, j
    } else {
      l += j
      if l >= mid {
        l -= mid
      } else {
        l = 0
      }
      r += j
      if r >= mid {
        r -= mid
      } else {
        r = 0
      }
    }
  }
  if l > MaxVol { l = MaxVol }
  if r > MaxVol { r = MaxVol }
  volume_left, volume_right = uint8(l), uint8(r)
  C.volCtrl (C.int(cdd), C.uchar(volume_left), C.uchar(volume_right))
  balance = Volume (Balance)
}


func init () {
//
  text[invalid] =    " invalid "
  text[playing] =    "   play  "
  text[paused] =     "  pause  "
  text[completed] =  "completed"
  text[fault] =      "  error  "
  text[nostatus] =   "no status"
  Ctrltext[Left] =    "left"
  Ctrltext[Right] =   "right"
  Ctrltext[Balance] = "balance"
  Ctrltext[All] =     "volume"
}
