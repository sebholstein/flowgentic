import type { FieldValues, UseFormReturn, SubmitHandler } from "react-hook-form";
import { FormProvider } from "react-hook-form";

interface FormProps<T extends FieldValues>
  extends Omit<React.ComponentProps<"form">, "onSubmit"> {
  form: UseFormReturn<T>;
  onSubmit: SubmitHandler<T>;
}

function Form<T extends FieldValues>({
  form,
  onSubmit,
  children,
  ...props
}: FormProps<T>) {
  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} {...props}>
        {children}
      </form>
    </FormProvider>
  );
}

export { Form };
