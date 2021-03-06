package robots

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/IBM/go-sdk-core/core"
	"github.com/algorithmiaio/algorithmia-go"
	"github.com/riqueemn/video-maker-go/entities"
	nlu "github.com/watson-developer-cloud/go-sdk/naturallanguageunderstandingv1"
	"gopkg.in/neurosnap/sentences.v1/english"
)


//Text -> struct do robô de texto
type Text struct {
}

//RobotProcess -> Sequência de processos do Robô
func (t *Text) RobotProcess() {
	var content = robotState.Load()

	fetchContentFromWikipedia(&content)
	sanitizeContent(&content)
	breakContentIntoSentences(&content)
	limitMaximumSentences(&content)
	fetchKeywordsOfAllSentences(&content)

	robotState.Save(content)

}

func myFunc(waitGroup *sync.WaitGroup) {
	time.Sleep(10 * time.Second)

	waitGroup.Done()
}

func fetchContentFromWikipedia(content *entities.Content) {

	var client = algorithmia.NewClient(secrets.APIKeyAlgorithmia, "")


	algo, _ := client.Algo("web/WikipediaParser/0.1.2?timeout=300")
	resp, _ := algo.Pipe(content.SearchTerm)
	response, _ := resp.(*algorithmia.AlgoResponse)

	wikiPediaContent := response.Result.(map[string]interface{})

	content.SourceContentOriginal = fmt.Sprintf("%v", wikiPediaContent["content"])

}

func sanitizeContent(content *entities.Content) {
	withoutBlankLines := removeBlankLines(content.SourceContentOriginal)
	withoutMarkdown := removeMarkdown(withoutBlankLines)
	withoutDatesInParenteses := removeDatesInParenteses(withoutMarkdown)
	//fmt.Println(withoutMarkdown)
	content.SourceContentSanitized = withoutDatesInParenteses
	//fmt.Println(len(withoutMarkdown))

}

func removeBlankLines(texto string) []string {
	allLines := strings.Split(texto, "\n")

	var withoutBlankLines []string
	for _, line := range allLines {
		if line != "" {
			withoutBlankLines = append(withoutBlankLines, line)
		}
	}

	return withoutBlankLines
}

func removeMarkdown(withoutBlankLines []string) string {
	var withoutMarkdown []string
	for _, line := range withoutBlankLines {
		if line[0] != '=' {
			withoutMarkdown = append(withoutMarkdown, line)
		}
	}

	return strings.Join(withoutMarkdown, " ")
}

func removeDatesInParenteses(withoutMarkdown string) string {
	var withoutDatesInParenteses string

	re := regexp.MustCompile(`[(]+[0-9A-Za-z,–\-./\t ]+[)]`)
	newLine := re.ReplaceAll([]byte(withoutMarkdown), []byte(""))

	re = regexp.MustCompile(`[\t ]+[\t ]`)
	newLine = re.ReplaceAll([]byte(newLine), []byte(" "))

	withoutDatesInParenteses = string(newLine)

	return withoutDatesInParenteses
}

func breakContentIntoSentences(content *entities.Content) {
	tokenizer, err := english.NewSentenceTokenizer(nil)
	if err != nil {
		panic(err)
	}

	sentences := tokenizer.Tokenize(content.SourceContentSanitized)
	content.Sentences = make([]entities.Sentence, len(sentences))
	for i, s := range sentences {
		content.Sentences[i].Text = s.Text
		content.Sentences[i].Keywords = nil
		content.Sentences[i].Images = nil
	}
}

func limitMaximumSentences(content *entities.Content) {
	content.Sentences = content.Sentences[0:7]
}

func fetchKeywordsOfAllSentences(content *entities.Content) {
	for i, sentence := range content.Sentences {
		content.Sentences[i].Keywords = fetchWatsonAndReturnKeyWords(sentence.Text)
	}
}

func fetchWatsonAndReturnKeyWords(sentence string) []string {
	authenticator := &core.IamAuthenticator{
		ApiKey: secrets.APIKeyWatson,
	}
	service, serviceErr := nlu.
		NewNaturalLanguageUnderstandingV1(&nlu.NaturalLanguageUnderstandingV1Options{
			URL:           "https://api.us-south.natural-language-understanding.watson.cloud.ibm.com/instances/f6c18e3b-0719-4eec-a61f-ae11232a0e4a",
			Version:       "2017-02-27",
			Authenticator: authenticator,
		})

	if serviceErr != nil {
		panic(serviceErr)
	}

	analyzeOptions := service.NewAnalyzeOptions(&nlu.Features{
		Keywords: &nlu.KeywordsOptions{},
	}).SetText(sentence)

	analyzeResult, _, responseErr := service.Analyze(analyzeOptions)

	if responseErr != nil {
		panic(responseErr)
	}

	var keywords []string
	if analyzeResult != nil {
		for _, keyword := range analyzeResult.Keywords {
			keywords = append(keywords, *keyword.Text)
		}
	}
	return keywords
}

func print(text []string) {
	for _, line := range text {

		t := fmt.Sprintf(";;%v;;", line)
		fmt.Println(t)
	}
}
