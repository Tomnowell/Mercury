package call

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/Tomnowell/Mercury/internal/registry"
)

func (controller *Controller) RunCLI() {
	log.Println("Enter Phone Number:")
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())

		switch {
		case input == "a":
			err := controller.AcceptCall()
			if err != nil {
				log.Println("Cannot accept call", err)
			}
		case input == "d":
			err := controller.DeclineCall()
			if err != nil {
				log.Println("Error when declining call", err)
			}
		case registry.IsPhoneNumber(input):
			number, err := registry.ParsePhoneNumber(input)

			if err != nil {
				log.Println("Invalid Number")
				continue
			}
			err = controller.Dial(number)
			if err != nil {
				log.Println("Error Could not dial number: ", number.String())
			}

		default:
			log.Println("Unknown Command")

		}
	}
}
