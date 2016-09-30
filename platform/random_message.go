package platform

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sync"
)

const (
	mediaBasePath = "./platform/media/"
)

// RandomMessage Struct to hold Random Message methods
type RandomMessage struct{}

type mediaCache struct {
	images map[string]bytes.Buffer
	mutex  sync.Mutex
}

// Plain returns a plain string message
func (rm RandomMessage) Plain() string {
	return Messages[rand.Intn(len(Messages))]
}

// Media will return the image
func (rm RandomMessage) Media() (name string, data *bytes.Buffer, ok bool) {
	name = MediaNames[rand.Intn(len(MediaNames))]
	data, ok = rm.loadImage(name)
	return
}

func (rm RandomMessage) loadImage(name string) (data *bytes.Buffer, ok bool) {
	cached, ok := MediaCache.images[name]
	if ok {
		return &cached, true
	}
	MediaCache.mutex.Lock()
	defer MediaCache.mutex.Unlock()
	mediaPath := fmt.Sprintf("%s%s", mediaBasePath, name)
	file, err := os.Open(mediaPath)
	if err != nil {
		panic(fmt.Sprintf("Image Not Found: %s", mediaPath))
	}
	defer file.Close()
	buffer := bytes.NewBuffer(nil)
	io.Copy(buffer, file)
	MediaCache.images[name] = *buffer
	return buffer, true
}

// MediaCache will hold loaded content
var MediaCache = mediaCache{make(map[string]bytes.Buffer), sync.Mutex{}}

// MediaNames will hold filenames of images in ./mediafolder
var MediaNames = [...]string{
	"test1.jpg",
	"test2.jpg",
	"test3.jpg",
	"test4.jpg",
	"test5.jpg",
	"test6.jpg",
	"test7.jpg",
	"test8.jpg",
	"test9.jpg",
	"test10.gif",
	"test11.gif",
	"test12.jpg",
	"test13.gif",
	"test14.jpg",
	"test16.jpg",
	"test17.png",
	"test18.png",
	"test20.gif",
	"test21.gif",
	"test22.jpg",
}

// Messages is a collection of randomly collected strings
var Messages = [...]string{
	"My horse loves lizards that chase dinosaurs!",
	"I enjoy spending time with the temporal rain forests' cheeses",
	"Oh my!!! I appear to have dropped my elephant!",
	"My hamster have itchy balls and OCD. I took him to see my therapist but she doesn't speak Hamsterian...",
	"My giraffe is quite offended by your hamsters... Being a hamster. Please tell him to stop immediately ",
	"Get out of my kitchen!!!!!!!! ",
	"Who you callin traffic pole?!? ",
	"NO. Don't tell me what to do! ",
	"He never listens does he? *grins evilly*",
	"Can a kangaroo jump higher than a house? Of course, a house doesn’t jump at all.",
	"Anton, do you think I’m a bad mother? My name is Paul.",
	"My dog used to chase people on a bike a lot. It got so bad, finally I had to take his bike away.",
	"The salt despairs the peanut above the satisfactory grace.",
	"Will an answering bread expire?",
	"The orchestra battles the duplicate.",
	"Why can't whatever hundred colleague burn in the custard?",
	"The graphic guilt airs an idiotic bread.",
	"The optic tower abides against the emotional back.",
	"The upstairs runs on top of the puzzle.",
	"Why won't a saga offend?",
	"A secretary polishes the whatever cream across a lad.",
	"A subtle dinner helps an immediate bell.",
	"Your plane intellectual towers without the darling partner.",
	"The weary vat glows.",
	"Underneath the coke quibbles a realized successor.",
	"The communal bug observes the detected workshop.",
	"A badge passes the hobby within the irrational arch.",
	"Why does the whistle shine before whatever hit recipe?",
	"The ally grows behind the orientated coast.",
	"In the boredom prevails the eighth spiritual.",
	"How can a graduate interest a courage?",
	"The spirit solos before the print tangent.",
	"His legendary aspect grows next to the heavy reward.",
	"The shed irony coughs.",
	"The pitfall denotes the consequent movie.",
	"A crisis starves underneath the proof!",
	"A cheerful radical compacts the bitmap above the super headache.",
	"When will the stair tap a disturbed hindsight?",
	"A hand renames the backbone.",
	"Your diverting space pops above the quality crossword.",
	"A pope decays before a dread.",
	"A numb evidence sighs before the sect.",
	"The darling nostalgia frowns.",
	"The internal cathedral groans.",
	"A flame acts next to a sermon!",
	"Should the treat pulp the dot?",
	"A fence calculates around our warehouse.",
	"Another rabid cigarette purges behind the brave load.",
	"The reform pretends without the classic box.",
	"The authorized plate strikes with another contract.",
}
